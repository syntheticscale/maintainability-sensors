package lsp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkLSPDidChange(b *testing.B) {
	dir := b.TempDir()
	filePath := filepath.Join(dir, "bench_test.go")
	content := "package main\n\nfunc benchmarkFunction() {\n\t// do nothing\n}\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	payload := fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file://%s","version":2},"contentChanges":[{"text":"%s"}]}}`, filePath, "package main\\n\\nfunc benchmarkFunction() {\\n\\t// something\\n}\\n")
	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(payload), payload)

	var in bytes.Buffer
	for i := 0; i < b.N; i++ {
		in.WriteString(msg)
	}

	b.ResetTimer()
	if err := Start(&in, io.Discard); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkDiskIOVsMemory_Disk(b *testing.B) {
	dir := b.TempDir()
	data := []byte("package main\n\nfunc benchmarkFunction() {\n\t// something\n}\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		func() {
			tmpFile, err := os.CreateTemp(dir, "lsp_temp_*.go")
			if err != nil {
				b.Fatal(err)
			}
			defer tmpFile.Close()
			defer os.Remove(tmpFile.Name())
			
			if _, err := tmpFile.Write(data); err != nil {
				b.Fatal(err)
			}
		}()
	}
}

func BenchmarkDiskIOVsMemory_Memory(b *testing.B) {
	data := []byte("package main\n\nfunc benchmarkFunction() {\n\t// something\n}\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := make([]byte, len(data))
		copy(buf, data)
		_ = buf
	}
}
