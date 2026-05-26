package protocol

const ProtocolVersion = 1

type FileContext struct {
	Path    string `json:"path"`
	Content []byte `json:"content,omitempty"`
}

type AnalyzeRequest struct {
	Version  int           `json:"version"`
	Language string        `json:"language"`
	Files    []FileContext `json:"files"`
}

type Violation struct {
	RuleName  string `json:"rule_name"`
	Value     int    `json:"value"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Message   string `json:"message"`
}

type AnalyzeResponse struct {
	Version int                    `json:"version"`
	Results map[string][]Violation `json:"results"`
	Error   string                 `json:"error,omitempty"`
}

type Handshake struct {
	SupportedLanguages []string `json:"supported_languages"`
}

type ParserRule struct {
	RuleName string
	Keys     []string
	Baseline int
}

const (
	BaselineComplexity          = 8
	BaselineCognitiveComplexity = 8
	BaselineFunctionLength      = 50
	BaselineArgumentCount       = 4
	BaselineFileLength          = 300
	BaselineCaseLength          = 10
)

const (
	FallbackEndLineOffset = 100
)

const (
	RuleComplexity          = "Complexity"
	RuleCognitiveComplexity = "CognitiveComplexity"
	RuleFunctionLength      = "FunctionLength"
	RuleArgumentCount       = "ArgumentCount"
	RuleCaseBlockLength     = "CaseBlockLength"
	RuleFileLength          = "FileLength"
)
