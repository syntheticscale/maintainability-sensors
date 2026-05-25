package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestFindFiles(t *testing.T) {
	tempDir := t.TempDir()

	filesToCreate := []string{
		"file1.go",
		"file2.js",
		"ignore.txt",
		filepath.Join(".git", "hidden.go"),
		filepath.Join("node_modules", "module.js"),
		filepath.Join("subdir", "file3.py"),
	}

	for _, f := range filesToCreate {
		fullPath := filepath.Join(tempDir, f)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(fullPath, []byte("content"), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("directory target", func(t *testing.T) {
		got, isDir, err := FindFiles(tempDir)
		if err != nil {
			t.Fatalf("FindFiles() unexpected error: %v", err)
		}
		if !isDir {
			t.Errorf("FindFiles() isDir = %v, want true", isDir)
		}

		wantFiles := []string{
			filepath.Join(tempDir, "file1.go"),
			filepath.Join(tempDir, "file2.js"),
			filepath.Join(tempDir, "subdir", "file3.py"),
		}
		
		var gotClean []string
		for _, f := range got {
			gotClean = append(gotClean, filepath.Clean(f))
		}
		var wantClean []string
		for _, f := range wantFiles {
			wantClean = append(wantClean, filepath.Clean(f))
		}
		
		sort.Strings(gotClean)
		sort.Strings(wantClean)

		if !reflect.DeepEqual(gotClean, wantClean) {
			t.Errorf("FindFiles() got files = %v, want %v", gotClean, wantClean)
		}
	})

	t.Run("file target", func(t *testing.T) {
		targetFile := filepath.Join(tempDir, "file1.go")
		got, isDir, err := FindFiles(targetFile)
		if err != nil {
			t.Fatalf("FindFiles() unexpected error: %v", err)
		}
		if isDir {
			t.Errorf("FindFiles() isDir = %v, want false", isDir)
		}
		if len(got) != 1 || got[0] != targetFile {
			t.Errorf("FindFiles() got = %v, want [%s]", got, targetFile)
		}
	})

	t.Run("non-existent target", func(t *testing.T) {
		_, _, err := FindFiles(filepath.Join(tempDir, "nonexistent"))
		if err == nil {
			t.Errorf("FindFiles() expected error for non-existent path")
		}
	})
}
