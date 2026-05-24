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

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	absPath = filepath.ToSlash(absPath)

	currentLayer := ""
	for layerName := range config.Layers {
		// e.g. path contains /api/
		if strings.Contains(absPath, "/"+layerName+"/") || strings.HasSuffix(absPath, "/"+layerName) || strings.HasPrefix(absPath, layerName+"/") {
			currentLayer = layerName
			break
		}
	}

	if currentLayer == "" {
		return violations, nil
	}

	allowedImports := config.Layers[currentLayer].AllowedImports
	allowedMap := make(map[string]bool)
	for _, imp := range allowedImports {
		allowedMap[imp] = true
	}

	if info, err := os.Stat(filePath); err == nil && (!info.Mode().IsRegular() || info.Size() > 2*1024*1024) {
		return violations, nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return violations, err
	}

	for _, imp := range f.Imports {
		if imp.Path == nil || imp.Path.Value == "" {
			continue
		}
		importPath := strings.Trim(imp.Path.Value, "\"")

		importedLayer := ""
		for layerName := range config.Layers {
			if strings.Contains(importPath, "/"+layerName+"/") || strings.HasSuffix(importPath, "/"+layerName) || importPath == layerName || strings.HasSuffix(importPath, "/"+layerName) {
				importedLayer = layerName
				break
			}
		}

		if importedLayer != "" && importedLayer != currentLayer && !allowedMap[importedLayer] {
			pos := fset.Position(imp.Pos())
			violations = append(violations, Violation{
				RuleName:  "DependencyBoundary",
				Message:   "Illegal import: layer '" + currentLayer + "' is not allowed to import layer '" + importedLayer + "'",
				StartLine: pos.Line,
				EndLine:   pos.Line,
				Value:     1,
			})
		}
	}

	return violations, nil
}
