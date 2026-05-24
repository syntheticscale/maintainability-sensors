package sensors

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	archConfigCache   = make(map[string]*ArchitectureConfig)
	archConfigCacheMu sync.RWMutex
)

func findArchitectureConfig(filePath string) *ArchitectureConfig {
	dir := filepath.Dir(filePath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	for {
		archConfigCacheMu.RLock()
		if cfg, exists := archConfigCache[absDir]; exists {
			archConfigCacheMu.RUnlock()
			return cfg
		}
		archConfigCacheMu.RUnlock()

		p := filepath.Join(absDir, ".sensors-architecture.yml")
		if info, err := os.Stat(p); err == nil && info.Mode().IsRegular() {
			cfg, err := ParseArchitectureConfig(p)
			if err == nil {
				archConfigCacheMu.Lock()
				archConfigCache[absDir] = cfg
				archConfigCacheMu.Unlock()
				return cfg
			}
		}

		parent := filepath.Dir(absDir)
		if parent == absDir {
			break
		}
		absDir = parent
	}

	archConfigCacheMu.Lock()
	archConfigCache[absDir] = nil
	archConfigCacheMu.Unlock()
	return nil
}

func CheckGoArchitecture(filePath string, config *ArchitectureConfig) ([]Violation, error) {
	var violations []Violation

	if config == nil || len(config.Layers) == 0 {
		return violations, nil
	}

	if info, err := os.Stat(filePath); err == nil && (!info.Mode().IsRegular() || info.Size() > 2*1024*1024) {
		return violations, nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return violations, err
	}

	var imports []ImportInfo
	for _, imp := range f.Imports {
		if imp.Path == nil || imp.Path.Value == "" {
			continue
		}
		importPath := strings.Trim(imp.Path.Value, "\"")
		pos := fset.Position(imp.Pos())
		imports = append(imports, ImportInfo{
			Path: importPath,
			Line: pos.Line,
		})
	}

	return CheckArchitectureDependencies(filePath, config, imports), nil
}
