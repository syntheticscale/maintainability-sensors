package protocol

import (
	"encoding/json"
	"testing"
)

func TestAnalyzeResponseTolerantReader(t *testing.T) {
	// A JSON payload with extra unknown fields to test "Tolerant Reader"
	jsonData := `{
		"results": {
			"file1.go": [
				{
					"rule_name": "Complexity",
					"value": 15,
					"start_line": 10,
					"end_line": 20,
					"message": "Too complex",
					"extra_field_ignore_me": "yes"
				}
			]
		},
		"error": "",
		"future_protocol_version": 2
	}`

	var resp AnalyzeResponse
	err := json.Unmarshal([]byte(jsonData), &resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal AnalyzeResponse: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("Expected empty error, got %q", resp.Error)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("Expected 1 result file, got %d", len(resp.Results))
	}

	violations := resp.Results["file1.go"]
	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.RuleName != "Complexity" {
		t.Errorf("Expected RuleName 'Complexity', got %q", v.RuleName)
	}
	if v.Value != 15 {
		t.Errorf("Expected Value 15, got %d", v.Value)
	}
}
