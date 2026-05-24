package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	TextDocumentSync int `json:"textDocumentSync,omitempty"`
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
