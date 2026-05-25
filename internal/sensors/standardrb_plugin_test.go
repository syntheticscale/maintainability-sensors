package sensors

import (
	"reflect"
	"testing"
)

func TestParseStandardRBMessages(t *testing.T) {
	tests := []struct {
		name  string
		files []StandardRBFile
		want  map[string][]Violation
	}{
		{
			name: "cyclomatic complexity",
			files: []StandardRBFile{
				{
					Path: "test.rb",
					Offenses: []StandardRBOffense{
						{
							CopName: "Metrics/CyclomaticComplexity",
							Message: "Cyclomatic complexity for method is too high. [10/7]",
							Location: StandardRBLocation{
								Line:     5,
								LastLine: 15,
							},
						},
					},
				},
			},
			want: map[string][]Violation{
				"test.rb": {
					{RuleName: RuleComplexity, Value: 10, StartLine: 5, EndLine: 15, Message: "Cyclomatic complexity for method is too high. [10/7]"},
				},
			},
		},
		{
			name: "method length",
			files: []StandardRBFile{
				{
					Path: "test.rb",
					Offenses: []StandardRBOffense{
						{
							CopName: "Metrics/MethodLength",
							Message: "Method has too many lines. [50/10]",
							Location: StandardRBLocation{
								Line:     20,
								LastLine: 70,
							},
						},
					},
				},
			},
			want: map[string][]Violation{
				"test.rb": {
					{RuleName: RuleFunctionLength, Value: 50, StartLine: 20, EndLine: 70, Message: "Method has too many lines. [50/10]"},
				},
			},
		},
		{
			name: "parameter list",
			files: []StandardRBFile{
				{
					Path: "test.rb",
					Offenses: []StandardRBOffense{
						{
							CopName: "Metrics/ParameterLists",
							Message: "Avoid parameter lists longer than 5 parameters. [6/5]",
							Location: StandardRBLocation{
								Line:     1,
								LastLine: 0,
							},
						},
					},
				},
			},
			want: map[string][]Violation{
				"test.rb": {
					{RuleName: RuleArgumentCount, Value: 6, StartLine: 1, EndLine: 1 + FallbackEndLineOffset, Message: "Avoid parameter lists longer than 5 parameters. [6/5]"},
				},
			},
		},
		{
			name: "unrelated cop",
			files: []StandardRBFile{
				{
					Path: "test.rb",
					Offenses: []StandardRBOffense{
						{
							CopName: "Style/StringLiterals",
							Message: "Prefer single-quoted strings...",
							Location: StandardRBLocation{
								Line:     1,
								LastLine: 1,
							},
						},
					},
				},
			},
			want: map[string][]Violation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStandardRBMessages(tt.files)
			if len(got) == 0 && len(tt.want) == 0 {
				return // both are empty
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStandardRBMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessStandardRBAnalyzeResult(t *testing.T) {
	tests := []struct {
		name      string
		exitCode  int
		result    StandardRBResult
		output    []byte
		wantErr   bool
		wantEmpty bool
	}{
		{
			name:     "success with offenses",
			exitCode: 1,
			result: StandardRBResult{
				Files: []StandardRBFile{
					{
						Path: "test.rb",
						Offenses: []StandardRBOffense{
							{
								CopName: "Metrics/CyclomaticComplexity",
								Message: "too high [10/7]",
							},
						},
					},
				},
			},
			wantErr:   false,
			wantEmpty: false,
		},
		{
			name:      "success without offenses",
			exitCode:  0,
			result:    StandardRBResult{},
			wantErr:   false,
			wantEmpty: true,
		},
		{
			name:     "crashed with exit code 1",
			exitCode: 1,
			result:   StandardRBResult{},
			output:   []byte("standardrb crashed"),
			wantErr:  true,
		},
		{
			name:     "unexpected exit code",
			exitCode: 2,
			result:   StandardRBResult{},
			output:   []byte("fatal error"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processStandardRBAnalyzeResult(tt.exitCode, tt.result, tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("processStandardRBAnalyzeResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.wantEmpty && len(got) != 0 {
					t.Errorf("processStandardRBAnalyzeResult() expected empty result, got %v", got)
				}
				if !tt.wantEmpty && len(got) == 0 {
					t.Errorf("processStandardRBAnalyzeResult() expected non-empty result, got empty")
				}
			}
		})
	}
}
