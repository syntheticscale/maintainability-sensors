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

	"github.com/paulolai/maintainability-sensors/sensors"
)

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
	promptsSB.WriteString("*Pass the following instructions directly to your AI Coding Agent (Cursor/Claude) to refactor the violations:*\n\n")

	for _, res := range results {
		if !res.ToolingDetected {
			continue
		}
		var filePrompts []string
		if res.Metrics.Complexity > 8 {
			filePrompts = append(filePrompts, fmt.Sprintf("  * Complexity is %d (Max 8). Nudge coding agent to extract nested conditionals into separate, single-responsibility helper functions.", res.Metrics.Complexity))
		}
		if res.Metrics.FunctionLength > 50 {
			filePrompts = append(filePrompts, fmt.Sprintf("  * Function lines is %d (Max 50). Nudge coding agent to modularize this block into separate functional components.", res.Metrics.FunctionLength))
		}
		if res.Metrics.ArgumentCount > 4 {
			filePrompts = append(filePrompts, fmt.Sprintf("  * Parameter count is %d (Max 4). Nudge coding agent to bundle parameters into a single structured configuration object.", res.Metrics.ArgumentCount))
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
		sb.WriteString("\n*If refactoring is impossible, instruct the agent to suppress the warning with standard inline annotations (e.g. // eslint-disable-next-line -- reason) to keep the baseline clean.*\n\n---\n\n")
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

// PostGitHubPRComment posts the markdown scorecard as a PR comment.
func PostGitHubPRComment(scorecard string) error {
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

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%s/comments", repo, prNumber)
	payload := map[string]string{
		"body": scorecard,
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
	req.Header.Set("X-GitHub-Api-Version: 2022-11-28", "")
	req.Header.Set("User-Agent", "Maintainability-Sensors-CLI")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
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

func getPRNumber() (string, error) {
	// 1. Try GITHUB_EVENT_PATH
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath != "" {
		data, err := os.ReadFile(eventPath)
		if err == nil {
			var event struct {
				PullRequest struct {
					Number int `json:"number"`
				} `json:"pull_request"`
			}
			if err := json.Unmarshal(data, &event); err == nil && event.PullRequest.Number > 0 {
				return fmt.Sprintf("%d", event.PullRequest.Number), nil
			}
		}
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
