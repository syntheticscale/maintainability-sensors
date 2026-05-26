package protocol

// FileContext represents a file to be analyzed.
type FileContext struct {
	Path    string `json:"path"`
	Content []byte `json:"content,omitempty"`
}

// AnalyzeRequest is sent over stdin to the plugin.
type AnalyzeRequest struct {
	Language string        `json:"language"`
	Files    []FileContext `json:"files"`
}

// Violation represents a single maintainability issue found in a file.
type Violation struct {
	RuleName  string `json:"rule_name"`
	Value     int    `json:"value"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Message   string `json:"message"`
}

// AnalyzeResponse is emitted by the plugin to stdout.
type AnalyzeResponse struct {
	Results map[string][]Violation `json:"results"`
	Error   string                 `json:"error,omitempty"`
}

// Handshake represents the initial message to verify the plugin capabilities.
type Handshake struct {
	SupportedLanguages []string `json:"supported_languages"`
}
