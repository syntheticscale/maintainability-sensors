package sensors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/syntheticscale/maintainability-sensors/internal/plugin/protocol"
)

// PluginRunner implements the Plugin interface by spawning a subprocess
// and communicating via the stdio JSON protocol.
type PluginRunner struct {
	PluginName string
	Command    string
	Args       []string
	Language   string
}

// Name returns the name of the plugin.
func (p *PluginRunner) Name() string {
	return p.PluginName
}

// Analyze sends an AnalyzeRequest to the plugin subprocess and parses its AnalyzeResponse.
func (p *PluginRunner) Analyze(files []FileContext) (map[string][]Violation, error) {
	reqBytes, err := p.buildRequest(files)
	if err != nil {
		return nil, err
	}

	stdout, err := p.executeCommand(reqBytes)
	if err != nil {
		return nil, err
	}

	return p.parseResponse(stdout)
}

func (p *PluginRunner) buildRequest(files []FileContext) ([]byte, error) {
	req := protocol.AnalyzeRequest{
		Language: p.Language,
		Files:    make([]protocol.FileContext, len(files)),
	}

	for i, f := range files {
		req.Files[i] = protocol.FileContext{
			Path:    f.Path,
			Content: f.Content,
		}
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return reqBytes, nil
}

func (p *PluginRunner) executeCommand(reqBytes []byte) ([]byte, error) {
	cmd := exec.Command(p.Command, p.Args...)
	cmd.Stdin = bytes.NewReader(reqBytes)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("plugin subprocess failed: %w, stderr: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func (p *PluginRunner) parseResponse(output []byte) (map[string][]Violation, error) {
	var resp protocol.AnalyzeResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse plugin response: %w, stdout: %s", err, string(output))
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("plugin returned error: %s", resp.Error)
	}

	result := make(map[string][]Violation)
	for filePath, protoViolations := range resp.Results {
		var violations []Violation
		for _, pv := range protoViolations {
			violations = append(violations, Violation{
				RuleName:  pv.RuleName,
				Value:     pv.Value,
				StartLine: pv.StartLine,
				EndLine:   pv.EndLine,
				Message:   pv.Message,
			})
		}
		result[filePath] = violations
	}

	return result, nil
}
