package sensors

import (
	"fmt"
	"path/filepath"
	"strings"
)

func sanitizePath(path string) (string, error) {
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("invalid path: contains null byte")
	}
	clean := filepath.Clean(path)
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("invalid path: traversal outside current directory denied")
	}
	return clean, nil
}

func sanitizeAndMapPaths(filePaths []string) ([]string, map[string]string, error) {
	validPaths := make([]string, 0, len(filePaths))
	originalPaths := make(map[string]string)

	for _, p := range filePaths {
		clean, err := sanitizePath(p)
		if err != nil {
			return nil, nil, err
		}
		abs, err := filepath.Abs(clean)
		if err == nil {
			originalPaths[abs] = p
		}
		originalPaths[clean] = p
		validPaths = append(validPaths, clean)
	}
	return validPaths, originalPaths, nil
}

func sanitizeAndMapFileContexts(files []FileContext) ([]FileContext, map[string]string, error) {
	validFiles := make([]FileContext, 0, len(files))
	originalPaths := make(map[string]string)

	for _, f := range files {
		clean, err := sanitizePath(f.Path)
		if err != nil {
			return nil, nil, err
		}
		abs, err := filepath.Abs(clean)
		if err == nil {
			originalPaths[abs] = f.Path
		}
		originalPaths[clean] = f.Path
		validFiles = append(validFiles, FileContext{Path: clean, Content: f.Content})
	}
	return validFiles, originalPaths, nil
}

func pathsMatch(p, absP, outPath string) bool {
	outAbs, _ := filepath.Abs(outPath)
	return outAbs == absP || outPath == p || filepath.Clean(outAbs) == filepath.Clean(p)
}
