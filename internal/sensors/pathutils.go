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

func pathsMatch(p, absP, outPath string) bool {
	outAbs, _ := filepath.Abs(outPath)
	return outAbs == absP || outPath == p || filepath.Clean(outAbs) == filepath.Clean(p)
}
