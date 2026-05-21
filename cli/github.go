package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/syntheticscale/maintainability-sensors/sensors"
)

const GitHubApiVersion = "2022-11-28"

// GenerateMarkdownScorecard generates a beautiful markdown scorecard of the scan results.
func GenerateMarkdownScorecard(results []sensors.OrchestratorResult) string {
	var sb strings.Builder

	sb.WriteString("# 📡 Maintainability Sensors Scorecard\n\n")
	sb.WriteString("## 📊 Scan Summary\n\n")
	sb.WriteString("| File | Language | Max Complexity | Max Func Lines | Max Params | Status |\n")
	sb.WriteString("| :--- | :--- | :---: | :---: | :---: | :---: |\n")

	for _, res := range results {
		fileBase := filepath.Base(res.FilePath)
		status := "ORCHESTRATED ✅"
		if !res.ToolingDetected {
			status = "BLIND ⚠️"
			sb.WriteString(fmt.Sprintf("| `%s` | %s | - | - | - | %s |\n", fileBase, strings.ToUpper(res.Language), status))
		} else {
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %d | %d | %d | %s |\n",
				fileBase, strings.ToUpper(res.Language), res.Metrics.Complexity, res.Metrics.FunctionLength, res.Metrics.ArgumentCount, status))
		}
	}

	sb.WriteString("\n---\n\n")

	// Self-Correction Prompts Section
	hasViolations := false
	var promptsSB strings.Builder
	promptsSB.WriteString("## ⚠️ AI Agent Self-Correction Prompts\n")
	promptsSB.WriteString("*AI WARNING: Refactor these violations:*\n\n")

	for _, res := range results {
		if !res.ToolingDetected {
			continue
		}
		var filePrompts []string
		if res.Metrics.Complexity > sensors.BaselineComplexity {
			filePrompts = append(filePrompts, fmt.Sprintf("  * Complexity is %d (Max %d). Nudge coding agent to extract nested conditionals into separate, single-responsibility helper functions.", res.Metrics.Complexity, sensors.BaselineComplexity))
		}
		if res.Metrics.FunctionLength > sensors.BaselineFunctionLength {
			filePrompts = append(filePrompts, fmt.Sprintf("  * Function lines is %d (Max %d). Nudge coding agent to modularize this block into separate functional components.", res.Metrics.FunctionLength, sensors.BaselineFunctionLength))
		}
		if res.Metrics.ArgumentCount > sensors.BaselineArgumentCount {
			filePrompts = append(filePrompts, fmt.Sprintf("  * Parameter count is %d (Max %d). Nudge coding agent to bundle parameters into a single structured configuration object.", res.Metrics.ArgumentCount, sensors.BaselineArgumentCount))
		}

		if len(filePrompts) > 0 {
			hasViolations = true
			promptsSB.WriteString(fmt.Sprintf("### 📄 `%s`:\n", filepath.Base(res.FilePath)))
			for _, p := range filePrompts {
				promptsSB.WriteString(p + "\n")
			}
			promptsSB.WriteString("\n")
		}
	}

	if hasViolations {
		sb.WriteString(promptsSB.String())
		sb.WriteString("\n*If refactoring is impossible, AI WARNING: suppress the warning with standard inline annotations (e.g. // eslint-disable-next-line -- reason) to keep the baseline clean.*\n\n---\n\n")
	}

	// Exceptions Section
	hasExceptions := false
	var excSB strings.Builder
	excSB.WriteString("## 🛠️ Exceptions Created by AI (Relaxed Constraints)\n")
	excSB.WriteString("*The following custom limits are set to relaxed values in the configuration. These relaxed thresholds must be manually verified by a human during code review.*\n\n")

	for _, res := range results {
		if len(res.Exceptions) > 0 {
			hasExceptions = true
			excSB.WriteString(fmt.Sprintf("### 📄 `%s`:\n", filepath.Base(res.FilePath)))
			for _, exc := range res.Exceptions {
				excSB.WriteString(fmt.Sprintf("  * **%s**: Configured Limit is %d (Standard Baseline is %d)\n", exc.RuleName, exc.ConfiguredVal, exc.BaselineVal))
			}
			excSB.WriteString("\n")
		}
	}

	if hasExceptions {
		sb.WriteString(excSB.String())
		sb.WriteString("> 💡 *“Looking at the exceptions AI created was a good point to start my code review.”* - Birgitta Böckeler\n\n")
	}

	return sb.String()
}

// WriteGitHubStepSummary writes the markdown scorecard to GITHUB_STEP_SUMMARY.
func WriteGitHubStepSummary(scorecard string) error {
	summaryPath := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryPath == "" {
		return nil
	}
	f, err := os.OpenFile(summaryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_STEP_SUMMARY file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(scorecard); err != nil {
		return fmt.Errorf("failed to write to GITHUB_STEP_SUMMARY: %w", err)
	}
	return nil
}

// PostGitHubReview posts inline PR review comments using the GitHub Pull Request Review API.
func PostGitHubReview(results []sensors.OrchestratorResult) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return fmt.Errorf("GITHUB_REPOSITORY environment variable is not set (expected 'owner/repo')")
	}

	prNumber, err := getPRNumber()
	if err != nil {
		return fmt.Errorf("failed to detect PR number: %w", err)
	}

	type prComment struct {
		Path string `json:"path"`
		Body string `json:"body"`
		Line int    `json:"line"`
	}
	var comments []prComment

	for _, res := range results {
		if !hasViolations(res) {
			continue
		}

		limitComplexity := sensors.BaselineComplexity
		limitLength := sensors.BaselineFunctionLength
		limitArgs := sensors.BaselineArgumentCount

		for _, exc := range res.Exceptions {
			if exc.RuleName == "Cyclomatic Complexity" {
				limitComplexity = exc.ConfiguredVal
			} else if exc.RuleName == "Function Length" {
				limitLength = exc.ConfiguredVal
			} else if exc.RuleName == "Argument Count" {
				limitArgs = exc.ConfiguredVal
			}
		}

		var filePrompts []string
		if res.Metrics.Complexity > limitComplexity {
			filePrompts = append(filePrompts, fmt.Sprintf("Complexity is %d (Max %d). Nudge coding agent to extract nested conditionals into separate, single-responsibility helper functions.", res.Metrics.Complexity, limitComplexity))
		}
		if res.Metrics.FunctionLength > limitLength {
			filePrompts = append(filePrompts, fmt.Sprintf("Function lines is %d (Max %d). Nudge coding agent to modularize this block into separate functional components.", res.Metrics.FunctionLength, limitLength))
		}
		if res.Metrics.ArgumentCount > limitArgs {
			filePrompts = append(filePrompts, fmt.Sprintf("Parameter count is %d (Max %d). Nudge coding agent to bundle parameters into a single structured configuration object.", res.Metrics.ArgumentCount, limitArgs))
		}

		if len(filePrompts) > 0 {
			body := strings.Join(filePrompts, "\n\n")
			relPath := res.FilePath
			if filepath.IsAbs(relPath) {
				wd, _ := os.Getwd()
				if rel, err := filepath.Rel(wd, relPath); err == nil {
					relPath = rel
				}
			}
			comments = append(comments, prComment{
				Path: filepath.ToSlash(relPath),
				Body: body,
				Line: 1,
			})
		}
	}

	if len(comments) == 0 {
		return nil
	}

	baseURL := os.Getenv("GITHUB_API_URL")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%s/reviews", baseURL, repo, prNumber)
	payload := map[string]interface{}{
		"body":     "Maintainability Sensors detected architectural decay.",
		"event":    "COMMENT",
		"comments": comments,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal comment payload: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", GitHubApiVersion)
	req.Header.Set("User-Agent", "Maintainability-Sensors-CLI")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned non-OK status: %s", resp.Status)
	}

	return nil
}

func getPRNumberFromEventPath() string {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return ""
	}
	info, err := os.Stat(eventPath)
	if err == nil && (!info.Mode().IsRegular() || info.Size() > 2*1024*1024) {
		return "" // skip if too large or not a regular file
	}
	data, err := os.ReadFile(eventPath)
	if err != nil {
		return ""
	}
	var event struct {
		PullRequest struct {
			Number int `json:"number"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &event); err == nil && event.PullRequest.Number > 0 {
		return fmt.Sprintf("%d", event.PullRequest.Number)
	}
	return ""
}

func getPRNumber() (string, error) {
	// 1. Try GITHUB_EVENT_PATH
	if num := getPRNumberFromEventPath(); num != "" {
		return num, nil
	}

	// 2. Try GITHUB_REF (e.g., refs/pull/123/merge)
	ref := os.Getenv("GITHUB_REF")
	if ref != "" {
		re := regexp.MustCompile(`^refs/pull/(\d+)/`)
		matches := re.FindStringSubmatch(ref)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("could not determine PR number from environment")
}
