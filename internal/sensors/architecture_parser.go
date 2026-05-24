package sensors

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ArchitectureConfig struct {
	Layers map[string]LayerConfig `yaml:"layers"`
}

type LayerConfig struct {
	AllowedImports []string `yaml:"allowed_imports"`
}

type ImportInfo struct {
	Path string
	Line int
}

func ParseArchitectureConfig(filePath string) (*ArchitectureConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var config ArchitectureConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func CheckArchitectureDependencies(filePath string, config *ArchitectureConfig, imports []ImportInfo) []Violation {
	var violations []Violation
	if config == nil || len(config.Layers) == 0 {
		return violations
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
		return violations
	}

	allowedImports := config.Layers[currentLayer].AllowedImports
	allowedMap := make(map[string]bool)
	for _, imp := range allowedImports {
		allowedMap[imp] = true
	}

	for _, imp := range imports {
		importPath := imp.Path

		importedLayer := ""
		for layerName := range config.Layers {
			if strings.Contains(importPath, "/"+layerName+"/") || strings.HasSuffix(importPath, "/"+layerName) || importPath == layerName {
				importedLayer = layerName
				break
			}
		}

		if importedLayer != "" && importedLayer != currentLayer && !allowedMap[importedLayer] {
			violations = append(violations, Violation{
				RuleName:  "DependencyBoundary",
				Message:   "Illegal import: layer '" + currentLayer + "' is not allowed to import layer '" + importedLayer + "'",
				StartLine: imp.Line,
				EndLine:   imp.Line,
				Value:     1,
			})
		}
	}

	return violations
}
