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
	result := make(map[string][]LineRange)

	// Get modified files using git diff
	ctxDiff, cancelDiff := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDiff()
	diffCmd := exec.CommandContext(ctxDiff, "git", "diff", targetBranch, "--unified=0", "--")
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

	scanner := bufio.NewScanner(stdoutPipe)
	var currentFile string

	for scanner.Scan() {
		line := scanner.Text()

		if matches := diffFileHeaderRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentFile = strings.TrimSpace(matches[1])
			// Initialize the slice for this file if it doesn't exist
			if _, exists := result[currentFile]; !exists {
				result[currentFile] = []LineRange{}
			}
			continue
		}

		if currentFile != "" {
			if matches := diffHunkHeaderRegex.FindStringSubmatch(line); len(matches) > 1 {
				startStr := matches[1]
				start, err := strconv.Atoi(startStr)
				if err != nil {
					continue
				}

				count := 1
				if len(matches) > 2 && matches[2] != "" {
					count, err = strconv.Atoi(matches[2])
					if err != nil {
						continue
					}
				}

				if count > 0 {
					end := start + count - 1
					result[currentFile] = append(result[currentFile], LineRange{Start: start, End: end})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
	        return nil, fmt.Errorf("error parsing diff output: %w", err)
	}

	err = diffCmd.Wait()
	if err != nil {
	        var exitErr *exec.ExitError
	        if errors.As(err, &exitErr) {
	                stderrStr := stderrBuf.String()
	                if exitErr.ExitCode() == 128 && strings.Contains(stderrStr, "ambiguous argument 'HEAD'") {
	                        // Fresh repo with no commits. We ignore this error.
	                } else {
	                        return nil, fmt.Errorf("git diff failed: %w (stderr: %s)", err, stderrStr)
	                }
	        } else {
	                return nil, fmt.Errorf("git diff failed: %w", err)
	        }
	}
	// Get untracked files
	ctxUntracked, cancelUntracked := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelUntracked()
	untrackedCmd := exec.CommandContext(ctxUntracked, "git", "ls-files", "--others", "--exclude-standard", "--")
	untrackedCmd.Dir = repoPath
	untrackedOut, err := untrackedCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	untrackedScanner := bufio.NewScanner(bytes.NewReader(untrackedOut))
	for untrackedScanner.Scan() {
		file := strings.TrimSpace(untrackedScanner.Text())
		if file != "" {
			fullPath := file
			if repoPath != "" && repoPath != "." {
				fullPath = filepath.Join(repoPath, file)
			}
			info, err := os.Stat(fullPath)
			if err == nil && (!info.Mode().IsRegular() || info.Size() > 2*1024*1024) {
				fmt.Fprintf(os.Stderr, "Warning: skipping large or non-regular untracked file %s\n", file)
				continue
			}
			result[file] = []LineRange{{Start: 1, End: 999999999}}
		}
	}

	if err := untrackedScanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing untracked files output: %w", err)
	}

	return result, nil
}
