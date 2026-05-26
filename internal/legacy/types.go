package legacy

import (
	"github.com/syntheticscale/maintainability-sensors/internal/plugin/protocol"
)

type Violation = protocol.Violation
type FileContext = protocol.FileContext

type ParserRule struct {
	RuleName string
	Keys     []string
	Baseline int
}
