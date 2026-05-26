package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/syntheticscale/maintainability-sensors/internal/legacy"
	"github.com/syntheticscale/maintainability-sensors/internal/plugin/protocol"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--handshake" {
		handshake := protocol.Handshake{
			SupportedLanguages: []string{"python", "ruby", "javascript", "typescript"},
		}
		out, _ := json.Marshal(handshake)
		fmt.Println(string(out))
		return
	}

	inputBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		emitError(fmt.Sprintf("failed to read stdin: %v", err))
		return
	}

	var req protocol.AnalyzeRequest
	if err := json.Unmarshal(inputBytes, &req); err != nil {
		emitError(fmt.Sprintf("failed to unmarshal request: %v", err))
		return
	}

	var plugins []interface {
		Analyze([]protocol.FileContext) (map[string][]protocol.Violation, error)
	}

	switch req.Language {
	case "python":
		plugins = append(plugins, &legacy.RuffPlugin{}, &legacy.PyLintPlugin{})
	case "ruby":
		plugins = append(plugins, &legacy.RuboCopPlugin{}, &legacy.StandardRBPlugin{})
	case "javascript", "typescript":
		plugins = append(plugins, &legacy.BiomePlugin{}, &legacy.ESLintPlugin{})
	}

	finalResults := make(map[string][]protocol.Violation)

	for _, p := range plugins {
		res, err := p.Analyze(req.Files)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin execution warning: %v\n", err)
			continue
		}
		for path, violations := range res {
			finalResults[path] = append(finalResults[path], violations...)
		}
	}

	resp := protocol.AnalyzeResponse{
		Results: finalResults,
	}

	out, err := json.Marshal(resp)
	if err != nil {
		emitError(fmt.Sprintf("failed to marshal response: %v", err))
		return
	}

	fmt.Println(string(out))
}

func emitError(msg string) {
	resp := protocol.AnalyzeResponse{Error: msg}
	out, _ := json.Marshal(resp)
	fmt.Println(string(out))
	os.Exit(1)
}
