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

func segmentsMatch(segments, layerSegments []string, startIndex int) bool {
	for j := 0; j < len(layerSegments); j++ {
		if segments[startIndex+j] != layerSegments[j] {
			return false
		}
	}
	return true
}

func matchesLayer(path, layerName string) bool {
	path = strings.Trim(filepath.ToSlash(path), "/")
	layerName = strings.Trim(filepath.ToSlash(layerName), "/")

	if path == layerName {
		return true
	}

	segments := strings.Split(path, "/")
	layerSegments := strings.Split(layerName, "/")

	if len(layerSegments) == 0 {
		return false
	}

	for i := 0; i <= len(segments)-len(layerSegments); i++ {
		if segmentsMatch(segments, layerSegments, i) {
			return true
		}
	}
	return false
}

func findLayer(path string, config *ArchitectureConfig) string {
	for layerName := range config.Layers {
		if matchesLayer(path, layerName) {
			return layerName
		}
	}
	return ""
}

func getViolation(currentLayer, importedLayer string, allowedMap map[string]bool, imp ImportInfo) *Violation {
	if importedLayer != "" && importedLayer != currentLayer && !allowedMap[importedLayer] {
		return &Violation{
			RuleName:  "DependencyBoundary",
			Message:   "Illegal import: layer '" + currentLayer + "' is not allowed to import layer '" + importedLayer + "'",
			StartLine: imp.Line,
			EndLine:   imp.Line,
			Value:     1,
		}
	}
	return nil
}

func getAbsPath(filePath string) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	return filepath.ToSlash(absPath)
}

func CheckArchitectureDependencies(filePath string, config *ArchitectureConfig, imports []ImportInfo) []Violation {
	var violations []Violation
	if config == nil || len(config.Layers) == 0 {
		return violations
	}

	absPath := getAbsPath(filePath)

	currentLayer := findLayer(absPath, config)
	if currentLayer == "" {
		return violations
	}

	allowedMap := make(map[string]bool)
	for _, imp := range config.Layers[currentLayer].AllowedImports {
		allowedMap[imp] = true
	}

	for _, imp := range imports {
		importedLayer := findLayer(imp.Path, config)
		if v := getViolation(currentLayer, importedLayer, allowedMap, imp); v != nil {
			violations = append(violations, *v)
		}
	}

	return violations
}
