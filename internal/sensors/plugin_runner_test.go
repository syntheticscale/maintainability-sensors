package sensors

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/syntheticscale/maintainability-sensors/internal/plugin/protocol"
)

func TestPluginRunnerAnalyzeSuccess(t *testing.T) {
	runner := &PluginRunner{
		PluginName: "TestPlugin",
		Command:    os.Args[0], // Use current test binary
		Args:       []string{"-test.run=TestHelperProcess", "--"},
		Language:   "python",
	}

	// Tell the helper process to act as the plugin
	os.Setenv("GO_WANT_HELPER_PROCESS", "1")
	defer os.Unsetenv("GO_WANT_HELPER_PROCESS")

	files := []FileContext{
		{Path: "test.py", Content: []byte("print('hello')")},
	}

	results, err := runner.Analyze(files)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result file, got %d", len(results))
	}

	violations, ok := results["test.py"]
	if !ok {
		t.Fatalf("Missing test.py in results")
	}

	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	expected := Violation{
		RuleName:  "TestRule",
		Value:     10,
		StartLine: 1,
		EndLine:   2,
		Message:   "Test message",
	}

	if !reflect.DeepEqual(violations[0], expected) {
		t.Errorf("Expected violation %+v, got %+v", expected, violations[0])
	}
}

func TestPluginRunnerAnalyzeError(t *testing.T) {
	runner := &PluginRunner{
		PluginName: "TestPlugin",
		Command:    os.Args[0], // Use current test binary
		Args:       []string{"-test.run=TestHelperProcess", "--"},
		Language:   "python",
	}

	// Tell the helper process to return an error
	os.Setenv("GO_WANT_HELPER_PROCESS", "2")
	defer os.Unsetenv("GO_WANT_HELPER_PROCESS")

	files := []FileContext{
		{Path: "test.py", Content: []byte("print('hello')")},
	}

	_, err := runner.Analyze(files)
	if err == nil {
		t.Fatalf("Expected analyze to fail but it succeeded")
	}

	expectedMsg := "plugin returned error: mock error from plugin"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "" {
		return
	}
	defer os.Exit(0)

	mode := os.Getenv("GO_WANT_HELPER_PROCESS")

	// Read stdin to ensure we can parse the AnalyzeRequest
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read stdin: %v\n", err)
		os.Exit(1)
	}

	var req protocol.AnalyzeRequest
	if err := json.Unmarshal(input, &req); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse stdin: %v\n", err)
		os.Exit(1)
	}

	if mode == "1" {
		// Success mode
		resp := protocol.AnalyzeResponse{
			Results: map[string][]protocol.Violation{
				"test.py": {
					{
						RuleName:  "TestRule",
						Value:     10,
						StartLine: 1,
						EndLine:   2,
						Message:   "Test message",
					},
				},
			},
		}
		out, _ := json.Marshal(resp)
		fmt.Fprint(os.Stdout, string(out))
	} else if mode == "2" {
		// Error mode
		resp := protocol.AnalyzeResponse{
			Error: "mock error from plugin",
		}
		out, _ := json.Marshal(resp)
		fmt.Fprint(os.Stdout, string(out))
	}
}
