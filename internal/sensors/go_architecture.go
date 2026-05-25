package sensors

import (
	"go/ast"
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

func checkCacheForArchConfig(absDir string) (*ArchitectureConfig, bool) {
	archConfigCacheMu.RLock()
	defer archConfigCacheMu.RUnlock()
	cfg, exists := archConfigCache[absDir]
	return cfg, exists
}

func parseAndCacheArchConfig(absDir, p string) (*ArchitectureConfig, bool) {
	if info, err := os.Stat(p); err == nil && info.Mode().IsRegular() {
		cfg, err := ParseArchitectureConfig(p)
		if err == nil {
			archConfigCacheMu.Lock()
			archConfigCache[absDir] = cfg
			archConfigCacheMu.Unlock()
			return cfg, true
		}
	}
	return nil, false
}

func cacheNilArchConfig(absDir string) {
	archConfigCacheMu.Lock()
	archConfigCache[absDir] = nil
	archConfigCacheMu.Unlock()
}

func findArchitectureConfig(filePath string) *ArchitectureConfig {
	dir := filepath.Dir(filePath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	for {
		if cfg, exists := checkCacheForArchConfig(absDir); exists {
			return cfg
		}

		p := filepath.Join(absDir, ".sensors-architecture.yml")
		if cfg, found := parseAndCacheArchConfig(absDir, p); found {
			return cfg
		}

		parent := filepath.Dir(absDir)
		if parent == absDir {
			break
		}
		absDir = parent
	}

	cacheNilArchConfig(absDir)
	return nil
}

func extractImports(fset *token.FileSet, f *ast.File) []ImportInfo {
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
	return imports
}

func CheckGoArchitecture(filePath string, config *ArchitectureConfig) ([]Violation, error) {
	var violations []Violation

	if config == nil || len(config.Layers) == 0 {
		return violations, nil
	}

	if info, err := os.Stat(filePath); err == nil && (!info.Mode().IsRegular() || info.Size() > MaxFileSize) {
		return violations, nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return violations, err
	}

	imports := extractImports(fset, f)

	return CheckArchitectureDependencies(filePath, config, imports), nil
}
