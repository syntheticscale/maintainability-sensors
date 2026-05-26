package lsp

//nolint // maintainability: highly cohesive test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestInitializeHandshake(t *testing.T) {
	reqJSON := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	payload := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(reqJSON), reqJSON)

	exitJSON := `{"jsonrpc":"2.0","method":"exit"}`
	exitPayload := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(exitJSON), exitJSON)

	inputBuffer := bytes.NewBufferString(payload + exitPayload)
	outputBuffer := bytes.NewBuffer(nil)

	err := Start(inputBuffer, outputBuffer)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	outStr := outputBuffer.String()
	if !bytes.Contains(outputBuffer.Bytes(), []byte("capabilities")) {
		t.Errorf("Expected response to contain 'capabilities', got: %s", outStr)
	}
}

func TestDidChangeDiagnostics(t *testing.T) {
	goCode := `package main

func highlyComplex() {
	if true {
		if true {
			if true {
				if true {
					if true {
						if true {
							if true {
								if true {
									if true {
										println("Complex")
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
`
	tmpFile, _ := os.CreateTemp("", "test_complex_*.go")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	uri := "file://" + tmpFile.Name()

	reqJSON := fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":%q,"version":2},"contentChanges":[{"text":%q}]}}`, uri, goCode)
	payload := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(reqJSON), reqJSON)

	exitJSON := `{"jsonrpc":"2.0","method":"exit"}`
	exitPayload := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(exitJSON), exitJSON)

	inputBuffer := bytes.NewBufferString(payload + exitPayload)
	outputBuffer := bytes.NewBuffer(nil)

	err := Start(inputBuffer, outputBuffer)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	outStr := outputBuffer.String()
	if !strings.Contains(outStr, "textDocument/publishDiagnostics") {
		t.Errorf("Expected response to contain 'textDocument/publishDiagnostics', got: %s", outStr)
	}
	if !strings.Contains(outStr, "Complexity") {
		t.Errorf("Expected diagnostic for Complexity, got: %s", outStr)
	}
}
