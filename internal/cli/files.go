package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func isSkippedDir(dirName string) bool {
	switch dirName {
	case "node_modules", ".git", "vendor", "bin", ".cache", ".venv", "venv", "env":
		return true
	}
	return false
}

func isValidExtension(ext string) bool {
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".py", ".go", ".rb", ".cs", ".java":
		return true
	}
	return false
}

func checkWalkDirPath(path string, absTargetDir string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	if resolvedPath, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolvedPath
	}
	if absPath != absTargetDir && !strings.HasPrefix(absPath, absTargetDir+string(filepath.Separator)) {
		return ""
	}
	return path
}

func processWalkDirFile(path string, d os.DirEntry, absTargetDir string) (string, error) {
	if d.IsDir() {
		if isSkippedDir(d.Name()) {
			return "", filepath.SkipDir
		}
		return "", nil
	}

	info, err := d.Info()
	if err != nil {
		logf(LogLevelWarn, "[WARNING] Cannot get info for %s: %v\n", path, err)
		return "", nil
	}
	if !info.Mode().IsRegular() {
		return "", nil
	}
	if info.Size() > sensors.MaxFileSize {
		logf(LogLevelWarn, "[WARNING] Skipping file %s: exceeds 2MB limit\n", path)
		return "", nil
	}

	if !isValidExtension(filepath.Ext(path)) {
		return "", nil
	}

	return checkWalkDirPath(path, absTargetDir), nil
}

func resolveSingleFile(cleanPath string, absTargetDir string) []string {
	var files []string
	absPath, err := filepath.Abs(cleanPath)
	if err == nil {
		if resolvedPath, err := filepath.EvalSymlinks(absPath); err == nil {
			absPath = resolvedPath
		}
		if strings.HasPrefix(absPath, absTargetDir+string(filepath.Separator)) || absPath == absTargetDir {
			files = append(files, cleanPath)
		}
	}
	return files
}

func FindFiles(targetPath string) ([]string, bool, error) {
	cleanPath := filepath.Clean(targetPath)

	absTargetDir, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, false, fmt.Errorf("[ERROR] Failed to get absolute path of target directory: %v", err)
	}

	if resolvedTargetDir, err := filepath.EvalSymlinks(absTargetDir); err == nil {
		absTargetDir = resolvedTargetDir
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, false, fmt.Errorf("[ERROR] Path does not exist: %s", targetPath)
	}

	isDir := info.IsDir()
	if !isDir {
		return resolveSingleFile(cleanPath, absTargetDir), false, nil
	}

	var files []string
	err = filepath.WalkDir(cleanPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			logf(LogLevelWarn, "[WARNING] Cannot access %s: %v\n", path, err)
			return nil
		}
		file, walkErr := processWalkDirFile(path, d, absTargetDir)
		if file != "" {
			files = append(files, file)
		}
		return walkErr
	})

	if err != nil {
		return nil, true, fmt.Errorf("[ERROR] Directory scan failed: %v", err)
	}

	return files, true, nil
}
