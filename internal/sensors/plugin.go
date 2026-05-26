package sensors

import (
	"sync"
)

type Violation struct {
	RuleName  string
	Value     int
	StartLine int
	EndLine   int
	Message   string
}

type FileContext struct {
	Path    string
	Content []byte
}

// Plugin defines the interface for language parsers and linters.
type Plugin interface {
	Name() string
	Analyze(files []FileContext) (map[string][]Violation, error)
}

// PluginRegistry manages available plugins.
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string][]Plugin
}

// NewPluginRegistry creates a new registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string][]Plugin),
	}
}

// Register adds a plugin for a specific language.
func (r *PluginRegistry) Register(language string, plugin Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[language] = append(r.plugins[language], plugin)
}

// GetPlugins returns all registered plugins for a given language.
func (r *PluginRegistry) GetPlugins(language string) []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.plugins[language]
}

// GlobalRegistry is the default registry for the application.
var GlobalRegistry = NewPluginRegistry()

func init() {
	legacyCmd := "./bin/legacy-plugin"

	// Register Javascript/Typescript plugins
	GlobalRegistry.Register("javascript", &PluginRunner{PluginName: "legacy-javascript", Command: legacyCmd, Language: "javascript"})
	GlobalRegistry.Register("typescript", &PluginRunner{PluginName: "legacy-typescript", Command: legacyCmd, Language: "typescript"})

	// Register Python plugins
	GlobalRegistry.Register("python", &PluginRunner{PluginName: "legacy-python", Command: legacyCmd, Language: "python"})

	// Register Ruby plugins
	GlobalRegistry.Register("ruby", &PluginRunner{PluginName: "legacy-ruby", Command: legacyCmd, Language: "ruby"})

	// Register Native plugins
	GlobalRegistry.Register("go", GoPlugin{})
}
