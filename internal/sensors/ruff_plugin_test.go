package sensors

//nolint // maintainability: highly cohesive test

import (
	"reflect"
	"testing"
)

func TestParseRuffMessages(t *testing.T) {
	tests := []struct {
		name string
		list []RuffMessage
		want map[string][]Violation
	}{
		{
			name: "complexity violation",
			list: []RuffMessage{
				{
					Filename: "test.py",
					Code:     "C901",
					Message:  "function is too complex (10 > 5)",
					Location: struct {
						Row int `json:"row"`
					}{Row: 10},
					EndLocation: struct {
						Row int `json:"row"`
					}{Row: 15},
				},
			},
			want: map[string][]Violation{
				"test.py": {
					{RuleName: RuleComplexity, Value: 10, StartLine: 10, EndLine: 15, Message: "function is too complex (10 > 5)"},
				},
			},
		},
		{
			name: "complexity without exact val",
			list: []RuffMessage{
				{
					Filename: "test.py",
					Code:     "C901",
					Message:  "is too complex",
					Location: struct {
						Row int `json:"row"`
					}{Row: 10},
					EndLocation: struct {
						Row int `json:"row"`
					}{Row: 15},
				},
			},
			want: map[string][]Violation{
				"test.py": {
					{RuleName: RuleComplexity, Value: 1, StartLine: 10, EndLine: 15, Message: "is too complex"},
				},
			},
		},
		{
			name: "function length violation",
			list: []RuffMessage{
				{
					Filename: "test.py",
					Code:     "PLR0915",
					Message:  "Too many statements (60 > 50)",
					Location: struct {
						Row int `json:"row"`
					}{Row: 20},
					EndLocation: struct {
						Row int `json:"row"`
					}{Row: 80},
				},
			},
			want: map[string][]Violation{
				"test.py": {
					{RuleName: RuleFunctionLength, Value: 60, StartLine: 20, EndLine: 80, Message: "Too many statements (60 > 50)"},
				},
			},
		},
		{
			name: "argument count violation",
			list: []RuffMessage{
				{
					Filename: "test.py",
					Code:     "PLR0913",
					Message:  "Too many arguments to function call (6 > 5)",
					Location: struct {
						Row int `json:"row"`
					}{Row: 5},
					EndLocation: struct {
						Row int `json:"row"`
					}{Row: 5},
				},
			},
			want: map[string][]Violation{
				"test.py": {
					{RuleName: RuleArgumentCount, Value: 6, StartLine: 5, EndLine: 5, Message: "Too many arguments to function call (6 > 5)"},
				},
			},
		},
		{
			name: "fallback end line",
			list: []RuffMessage{
				{
					Filename: "test.py",
					Code:     "C901",
					Message:  "too complex (8 > 5)",
					Location: struct {
						Row int `json:"row"`
					}{Row: 10},
					EndLocation: struct {
						Row int `json:"row"`
					}{Row: 0},
				},
			},
			want: map[string][]Violation{
				"test.py": {
					{RuleName: RuleComplexity, Value: 8, StartLine: 10, EndLine: 10 + FallbackEndLineOffset, Message: "too complex (8 > 5)"},
				},
			},
		},
		{
			name: "unrelated code",
			list: []RuffMessage{
				{
					Filename: "test.py",
					Code:     "E501",
					Message:  "Line too long (90 > 88 characters)",
					Location: struct {
						Row int `json:"row"`
					}{Row: 10},
				},
			},
			want: map[string][]Violation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRuffMessages(tt.list)
			if len(got) == 0 && len(tt.want) == 0 {
				return // both are empty
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRuffMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}
