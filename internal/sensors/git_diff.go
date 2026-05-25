package sensors

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// LineRange represents a range of modified lines in a file.
type LineRange struct {
	Start int
	End   int
}

var (
	diffFileHeaderRegex = regexp.MustCompile(`^\+\+\+ b/(.*)$`)
	diffHunkHeaderRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
)

// GetModifiedLines runs git diff against the target branch and returns a map
// of file paths to the line ranges that were added or modified.
func GetModifiedLines(targetBranch string, repoPath string) (map[string][]LineRange, error) {
	if targetBranch == "" {
		targetBranch = "HEAD"
	}
	if strings.HasPrefix(targetBranch, "-") {
		return nil, fmt.Errorf("invalid target branch: cannot start with '-'")
	}

	result, err := getGitDiffLines(targetBranch, repoPath)
	if err != nil {
		return nil, err
	}

	if err := addUntrackedFiles(repoPath, result); err != nil {
		return nil, err
	}

	return result, nil
}

func getGitDiffLines(targetBranch, repoPath string) (map[string][]LineRange, error) {
	ctxDiff, cancelDiff := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDiff()
	diffCmd := exec.CommandContext(ctxDiff, "git", "-c", "core.quotepath=false", "diff", targetBranch, "--unified=0", "--")
	diffCmd.Dir = repoPath

	stdoutPipe, err := diffCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("git diff stdout pipe failed: %w", err)
	}

	var stderrBuf bytes.Buffer
	diffCmd.Stderr = &stderrBuf

	if err := diffCmd.Start(); err != nil {
		return nil, fmt.Errorf("git diff start failed: %w", err)
	}

	result, err := parseGitDiffOutput(bufio.NewScanner(stdoutPipe))
	if err != nil {
		return nil, err
	}

	err = diffCmd.Wait()
	if err != nil {
		return nil, handleGitDiffError(err, stderrBuf.String())
	}

	return result, nil
}

func handleGitDiffError(err error, stderrStr string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 128 && strings.Contains(stderrStr, "ambiguous argument 'HEAD'") {
			// Fresh repo with no commits. We ignore this error.
			return nil
		}
		return fmt.Errorf("git diff failed: %w (stderr: %s)", err, stderrStr)
	}
	return fmt.Errorf("git diff failed: %w", err)
}

func parseGitDiffOutput(scanner *bufio.Scanner) (map[string][]LineRange, error) {
	result := make(map[string][]LineRange)
	var currentFile string

	for scanner.Scan() {
		currentFile = processDiffLine(scanner.Text(), currentFile, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing diff output: %w", err)
	}

	return result, nil
}

func processDiffLine(line, currentFile string, result map[string][]LineRange) string {
	if matches := diffFileHeaderRegex.FindStringSubmatch(line); len(matches) > 1 {
		file := parseDiffFileHeader(matches[1])
		if _, exists := result[file]; !exists {
			result[file] = []LineRange{}
		}
		return file
	}

	if currentFile == "" {
		return currentFile
	}

	matches := diffHunkHeaderRegex.FindStringSubmatch(line)
	if len(matches) <= 1 {
		return currentFile
	}

	if lr, ok := parseDiffHunk(matches); ok {
		result[currentFile] = append(result[currentFile], lr)
	}

	return currentFile
}

func parseDiffFileHeader(raw string) string {
	currentFile := strings.TrimSpace(raw)
	if strings.HasPrefix(currentFile, `"`) && strings.HasSuffix(currentFile, `"`) {
		if unquoted, err := strconv.Unquote(currentFile); err == nil {
			return unquoted
		}
	}
	return currentFile
}

func parseDiffHunk(matches []string) (LineRange, bool) {
	startStr := matches[1]
	start, err := strconv.Atoi(startStr)
	if err != nil {
		return LineRange{}, false
	}

	count := 1
	if len(matches) > 2 && matches[2] != "" {
		count, err = strconv.Atoi(matches[2])
		if err != nil {
			return LineRange{}, false
		}
	}

	if count > 0 {
		return LineRange{Start: start, End: start + count - 1}, true
	}
	return LineRange{}, false
}

func addUntrackedFiles(repoPath string, result map[string][]LineRange) error {
	ctxUntracked, cancelUntracked := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelUntracked()
	untrackedCmd := exec.CommandContext(ctxUntracked, "git", "ls-files", "-z", "--others", "--exclude-standard", "--")
	untrackedCmd.Dir = repoPath
	untrackedOut, err := untrackedCmd.Output()
	if err != nil {
		return fmt.Errorf("git ls-files failed: %w", err)
	}

	for _, file := range strings.Split(string(untrackedOut), "\x00") {
		if file != "" {
			processUntrackedFile(file, repoPath, result)
		}
	}
	return nil
}

func processUntrackedFile(file, repoPath string, result map[string][]LineRange) {
	fullPath := file
	if repoPath != "" && repoPath != "." {
		fullPath = filepath.Join(repoPath, file)
	}
	info, err := os.Stat(fullPath)
	if err == nil && (!info.Mode().IsRegular() || info.Size() > MaxFileSize) {
		fmt.Fprintf(os.Stderr, "Warning: skipping large or non-regular untracked file %s\n", file)
		return
	}
	result[file] = []LineRange{{Start: 1, End: UntrackedFileEndLine}}
}
