package sensors

import "sync"

type Violation struct {
	RuleName  string
	Value     int
	StartLine int
	EndLine   int
	Message   string
}

// Plugin defines the interface for language parsers and linters.
type Plugin interface {
	Name() string
	Analyze(filePaths []string) (map[string][]Violation, error)
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
	// Register Javascript/Typescript plugins
	GlobalRegistry.Register("javascript", ESLintPlugin{})
	GlobalRegistry.Register("typescript", ESLintPlugin{})
	GlobalRegistry.Register("javascript", BiomePlugin{})
	GlobalRegistry.Register("typescript", BiomePlugin{})

	// Register Python plugins
	GlobalRegistry.Register("python", PythonTreeSitterPlugin{})
	GlobalRegistry.Register("python", PyLintPlugin{})
	GlobalRegistry.Register("python", RuffPlugin{})

	// Register Ruby plugins
	GlobalRegistry.Register("ruby", RuboCopPlugin{})
	GlobalRegistry.Register("ruby", StandardRBPlugin{})

	// Register Native plugins
	GlobalRegistry.Register("go", GoPlugin{})
	GlobalRegistry.Register("csharp", CSharpPlugin{})
	GlobalRegistry.Register("java", JavaPlugin{})
}
