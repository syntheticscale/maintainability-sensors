package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

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

// jsonRPCWriter serializes JSON-RPC messages to an io.Writer with
// mutex protection, ensuring atomic per-message writes even under
// concurrent callers.
type jsonRPCWriter struct {
	out io.Writer
	mu  sync.Mutex
}

func newJSONRPCWriter(out io.Writer) *jsonRPCWriter {
	return &jsonRPCWriter{out: out}
}

func (w *jsonRPCWriter) sendNotification(notif Notification) error {
	resBytes, err := json.Marshal(notif)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(resBytes))

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.out.Write([]byte(header)); err != nil {
		return err
	}
	_, err = w.out.Write(resBytes)
	return err
}

func (w *jsonRPCWriter) sendResponse(res Response) error {
	resBytes, err := json.Marshal(res)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(resBytes))

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.out.Write([]byte(header)); err != nil {
		return err
	}
	_, err = w.out.Write(resBytes)
	return err
}

func getLimitForRule(ruleName string, exceptions []sensors.RelaxedLimit) int {
	for _, e := range exceptions {
		if e.RuleName == ruleName {
			return e.ConfiguredVal
		}
	}
	switch ruleName {
	case sensors.RuleComplexity:
		return sensors.BaselineComplexity
	case sensors.RuleCognitiveComplexity:
		return sensors.BaselineCognitiveComplexity
	case sensors.RuleFunctionLength:
		return sensors.BaselineFunctionLength
	case sensors.RuleArgumentCount:
		return sensors.BaselineArgumentCount
	case sensors.RuleCaseBlockLength:
		return sensors.BaselineCaseLength
	}
	return sensors.FallbackLimit // fallback
}

func StartServer() error {
	return Start(os.Stdin, os.Stdout)
}

func parseContentLength(line string, currentLen int) (int, error) {
	if !strings.HasPrefix(line, "Content-Length: ") {
		return currentLen, nil
	}
	lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length: "))
	return strconv.Atoi(lengthStr)
}

func readHeader(reader *bufio.Reader) (int, error) {
	var contentLength int
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		contentLength, err = parseContentLength(line, contentLength)
		if err != nil {
			return 0, err
		}
	}
	return contentLength, nil
}

func handleInitialize(req Request, writer *jsonRPCWriter) {
	res := Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync: 1, // Full sync
			},
		},
	}
	writer.sendResponse(res)
}

func createDiagnostics(violations []sensors.Violation, exceptions []sensors.RelaxedLimit) []Diagnostic {
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
	return diags
}

func handleDidChange(req Request, writer *jsonRPCWriter) {
	var didChangeParams DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params, &didChangeParams); err != nil {
		return
	}

	if len(didChangeParams.ContentChanges) == 0 {
		return
	}
	newText := didChangeParams.ContentChanges[0].Text
	uri := didChangeParams.TextDocument.URI

	filePath := strings.TrimPrefix(uri, "file://")
	lang := sensors.DetectLanguage(filePath)
	if lang == "" {
		return
	}

	fileCtx := sensors.FileContext{
		Path:    filePath,
		Content: []byte(newText),
	}

	metricsMap, err := sensors.ScanDeltaBatch([]sensors.FileContext{fileCtx}, map[string]string{fileCtx.Path: fileCtx.Path}, lang)
	if err != nil {
		return
	}

	violations := metricsMap[fileCtx.Path]

	var exceptions []sensors.RelaxedLimit
	if anchor, parser := sensors.DetectConfigAndParser(fileCtx.Path, lang); anchor != "" {
		exceptions = sensors.DetectRelaxedLimits(anchor, parser)
	}

	diags := createDiagnostics(violations, exceptions)

	notif := Notification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diags,
		},
	}
	writer.sendNotification(notif)
}

func processRequest(req Request, writer *jsonRPCWriter) bool {
	switch req.Method {
	case "initialize":
		handleInitialize(req, writer)
	case "textDocument/didChange":
		handleDidChange(req, writer)
	case "exit":
		return true
	}
	return false
}

func processNextMessage(reader *bufio.Reader, writer *jsonRPCWriter) (bool, error) {
	contentLength, err := readHeader(reader)
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, err
	}

	if contentLength == 0 {
		return true, nil
	}

	content := make([]byte, contentLength)
	if _, err = io.ReadFull(reader, content); err != nil {
		return false, err
	}

	var req Request
	if err := json.Unmarshal(content, &req); err != nil {
		return true, nil
	}

	if processRequest(req, writer) {
		return false, nil
	}
	return true, nil
}

func Start(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	writer := newJSONRPCWriter(out)

	for {
		continueReading, err := processNextMessage(reader, writer)
		if err != nil {
			return err
		}
		if !continueReading {
			return nil
		}
	}
}
