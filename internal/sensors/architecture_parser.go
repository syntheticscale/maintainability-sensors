package sensors

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ArchitectureConfig struct {
	Dir    string
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

func matchesLayer(path, layerName string) int {
	path = strings.Trim(filepath.ToSlash(path), "/")
	layerName = strings.Trim(filepath.ToSlash(layerName), "/")

	if path == layerName {
		return 0
	}

	segments := strings.Split(path, "/")
	layerSegments := strings.Split(layerName, "/")

	if len(layerSegments) == 0 {
		return -1
	}

	for i := 0; i <= len(segments)-len(layerSegments); i++ {
		if segmentsMatch(segments, layerSegments, i) {
			return i
		}
	}
	return -1
}

func isBetterLayerMatch(idx, layerLen, bestIndex, bestLen int) bool {
	if bestIndex == -1 {
		return true
	}
	if idx < bestIndex {
		return true
	}
	return idx == bestIndex && layerLen > bestLen
}

func findLayer(path string, config *ArchitectureConfig) string {
	bestLayer := ""
	bestIndex := -1
	bestLen := -1

	for layerName := range config.Layers {
		idx := matchesLayer(path, layerName)
		if idx == -1 {
			continue
		}
		
		layerLen := len(strings.Split(strings.Trim(filepath.ToSlash(layerName), "/"), "/"))
		if isBetterLayerMatch(idx, layerLen, bestIndex, bestLen) {
			bestLayer = layerName
			bestIndex = idx
			bestLen = layerLen
		}
	}
	return bestLayer
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
