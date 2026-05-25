package sensors

import (
	"reflect"
	"testing"
)

func TestParseBiomeMessages(t *testing.T) {
	tests := []struct {
		name        string
		diagnostics []BiomeDiagnostic
		want        map[string][]Violation
	}{
		{
			name: "complexity violation",
			diagnostics: []BiomeDiagnostic{
				{
					Category: "complexity/noExcessiveCognitiveComplexity",
					Location: BiomeLocation{
						Path: struct {
							File string `json:"file"`
						}{File: "test.js"},
						Span: struct {
							Start int `json:"start"`
							End   int `json:"end"`
						}{Start: 10, End: 15},
					},
					Description: "cognitive complexity of 20 exceeds limit",
				},
			},
			want: map[string][]Violation{
				"test.js": {
					{RuleName: RuleComplexity, Value: 20, StartLine: 10, EndLine: 15, Message: "cognitive complexity of 20 exceeds limit"},
				},
			},
		},
		{
			name: "max parameters violation",
			diagnostics: []BiomeDiagnostic{
				{
					Category: "complexity/noExcessiveParameters",
					Location: BiomeLocation{
						Path: struct {
							File string `json:"file"`
						}{File: "test.js"},
						Span: struct {
							Start int `json:"start"`
							End   int `json:"end"`
						}{Start: 20, End: 0},
					},
					Description: "too many parameters (5)",
				},
			},
			want: map[string][]Violation{
				"test.js": {
					{RuleName: RuleArgumentCount, Value: 5, StartLine: 20, EndLine: 20 + FallbackEndLineOffset, Message: "too many parameters (5)"},
				},
			},
		},
		{
			name: "unrelated violation",
			diagnostics: []BiomeDiagnostic{
				{
					Category: "style/useConst",
					Location: BiomeLocation{
						Path: struct {
							File string `json:"file"`
						}{File: "test.js"},
						Span: struct {
							Start int `json:"start"`
							End   int `json:"end"`
						}{Start: 10, End: 15},
					},
					Description: "use const instead of let",
				},
			},
			want: map[string][]Violation{}, // changed to map[string][]Violation{}
		},
		{
			name: "empty path",
			diagnostics: []BiomeDiagnostic{
				{
					Category: "complexity/noExcessiveCognitiveComplexity",
					Location: BiomeLocation{
						Span: struct {
							Start int `json:"start"`
							End   int `json:"end"`
						}{Start: 10, End: 15},
					},
					Description: "complexity 10",
				},
			},
			want: map[string][]Violation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBiomeMessages(tt.diagnostics)
			if len(got) == 0 && len(tt.want) == 0 {
				return // both are empty
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseBiomeMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessBiomeAnalyzeResult(t *testing.T) {
	tests := []struct {
		name      string
		exitCode  int
		result    BiomeResult
		output    []byte
		wantErr   bool
		wantEmpty bool
	}{
		{
			name:     "success with diagnostics",
			exitCode: 1,
			result: BiomeResult{
				Diagnostics: []BiomeDiagnostic{
					{
						Category: "complexity/noExcessiveCognitiveComplexity",
						Location: BiomeLocation{
							Path: struct {
								File string `json:"file"`
							}{File: "test.js"},
							Span: struct {
								Start int `json:"start"`
								End   int `json:"end"`
							}{Start: 10, End: 15},
						},
						Description: "complexity of 20",
					},
				},
			},
			wantErr:   false,
			wantEmpty: false,
		},
		{
			name:      "success without diagnostics",
			exitCode:  0,
			result:    BiomeResult{},
			wantErr:   false,
			wantEmpty: true,
		},
		{
			name:     "crashed with exit code 1",
			exitCode: 1,
			result:   BiomeResult{},
			output:   []byte("biome crashed"),
			wantErr:  true,
		},
		{
			name:     "unexpected exit code",
			exitCode: 2,
			result:   BiomeResult{},
			output:   []byte("biome fatal error"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processBiomeAnalyzeResult(tt.exitCode, tt.result, tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("processBiomeAnalyzeResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.wantEmpty && len(got) != 0 {
					t.Errorf("processBiomeAnalyzeResult() expected empty result, got %v", got)
				}
				if !tt.wantEmpty && len(got) == 0 {
					t.Errorf("processBiomeAnalyzeResult() expected non-empty result, got empty")
				}
			}
		})
	}
}
