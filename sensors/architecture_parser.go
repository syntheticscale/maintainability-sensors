package sensors

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ArchitectureConfig struct {
	Layers map[string]LayerConfig `yaml:"layers"`
}

type LayerConfig struct {
	AllowedImports []string `yaml:"allowed_imports"`
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
