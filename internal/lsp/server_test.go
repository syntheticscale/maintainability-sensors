package lsp

import (
	"bytes"
	"fmt"
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
