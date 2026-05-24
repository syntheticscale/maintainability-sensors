package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int        `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

const (
	DiagnosticSeverityError       = 1
	DiagnosticSeverityWarning     = 2
	DiagnosticSeverityInformation = 3
	DiagnosticSeverityHint        = 4
)

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	TextDocumentSync int `json:"textDocumentSync,omitempty"`
}

func getLimitForRule(ruleName string, exceptions []sensors.RelaxedLimit) int {
	for _, e := range exceptions {
		if e.RuleName == ruleName {
			return e.ConfiguredVal
		}
	}
	switch ruleName {
	case "Complexity":
		return sensors.BaselineComplexity
	case "CognitiveComplexity":
		return sensors.BaselineCognitiveComplexity
	case "FunctionLength":
		return sensors.BaselineFunctionLength
	case "ArgumentCount":
		return sensors.BaselineArgumentCount
	case "CaseBlockLength":
		return sensors.BaselineCaseLength
	}
	return 999999 // fallback
}

func sendNotification(out io.Writer, notif Notification) error {
	resBytes, err := json.Marshal(notif)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(resBytes))
	_, err = out.Write([]byte(header))
	if err != nil {
		return err
	}
	_, err = out.Write(resBytes)
	return err
}

func StartServer() error {
	return Start(os.Stdin, os.Stdout)
}

func Start(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)

	for {
		var contentLength int
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length: ") {
				lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length: "))
				contentLength, err = strconv.Atoi(lengthStr)
				if err != nil {
					return err
				}
			}
		}

		if contentLength == 0 {
			continue
		}

		content := make([]byte, contentLength)
		_, err := io.ReadFull(reader, content)
		if err != nil {
			return err
		}

		var req Request
		if err := json.Unmarshal(content, &req); err != nil {
			continue
		}

		if req.Method == "initialize" {
			res := Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: InitializeResult{
					Capabilities: ServerCapabilities{
						TextDocumentSync: 1, // Full sync
					},
				},
			}
			sendResponse(out, res)
		} else if req.Method == "textDocument/didChange" {
			var didChangeParams DidChangeTextDocumentParams
			if err := json.Unmarshal(req.Params, &didChangeParams); err != nil {
				continue
			}

			if len(didChangeParams.ContentChanges) == 0 {
				continue
			}
			newText := didChangeParams.ContentChanges[0].Text
			uri := didChangeParams.TextDocument.URI
			
			filePath := strings.TrimPrefix(uri, "file://")
			lang := sensors.DetectLanguage(filePath)
			if lang == "" {
				continue
			}

			dir := filepath.Dir(filePath)
			ext := filepath.Ext(filePath)
			
			tmpFile, err := os.CreateTemp(dir, fmt.Sprintf(".lsp_temp_*%s", ext))
			if err != nil {
				continue
			}
			tmpName := tmpFile.Name()
			tmpFile.Write([]byte(newText))
			tmpFile.Close()

			func(tmpPath, origPath, uri, lang string) {
				defer os.Remove(tmpPath)

				metricsMap, err := sensors.ScanDeltaBatch([]string{tmpPath}, map[string]string{tmpPath: origPath}, lang)
				if err != nil {
					return
				}

				violations := metricsMap[origPath]
				
				var exceptions []sensors.RelaxedLimit
				if anchor, parser := sensors.DetectConfigAndParser(origPath, lang); anchor != "" {
					exceptions = sensors.DetectRelaxedLimits(anchor, parser)
				}
				
				var diags []Diagnostic
				for _, v := range violations {
					limit := getLimitForRule(v.RuleName, exceptions)
					if v.Value > limit {
						diags = append(diags, Diagnostic{
							Range: Range{
								Start: Position{Line: v.StartLine - 1, Character: 0},
								End:   Position{Line: v.EndLine - 1, Character: 100},
							},
							Severity: DiagnosticSeverityWarning,
							Source:   "maintainability-sensors",
							Message:  fmt.Sprintf("%s: %s (Value: %d, Limit: %d)", v.RuleName, v.Message, v.Value, limit),
						})
					}
				}

				if diags == nil {
					diags = []Diagnostic{}
				}

				notif := Notification{
					JSONRPC: "2.0",
					Method:  "textDocument/publishDiagnostics",
					Params: PublishDiagnosticsParams{
						URI:         uri,
						Diagnostics: diags,
					},
				}
				sendNotification(out, notif)
			}(tmpName, filePath, uri, lang)
		} else if req.Method == "exit" {
			return nil
		}
	}
}

func sendResponse(out io.Writer, res Response) error {
	resBytes, err := json.Marshal(res)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(resBytes))
	_, err = out.Write([]byte(header))
	if err != nil {
		return err
	}
	_, err = out.Write(resBytes)
	return err
}
