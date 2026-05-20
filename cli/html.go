package cli

import (
	"fmt"
	"html"
	"path/filepath"
	"strings"

	"github.com/paulolai/maintainability-sensors/sensors"
)

// GenerateHTMLScorecard compiles a self-contained, beautifully styled, dark-themed HTML report.
func GenerateHTMLScorecard(results []sensors.OrchestratorResult) string {
	// Calculate quick stats
	totalFiles := len(results)
	orchestratedCount := 0
	blindCount := 0
	totalViolations := 0
	totalExceptions := 0

	var tableRows strings.Builder
	var promptCards strings.Builder
	var exceptionCards strings.Builder

	for _, res := range results {
		fileBase := filepath.Base(res.FilePath)
		if res.ToolingDetected {
			orchestratedCount++
			totalExceptions += len(res.Exceptions)

			// Check for violations in this file
			hasViolations := false
			var filePrompts []string
			if res.Metrics.Complexity > 8 {
				hasViolations = true
				totalViolations++
				filePrompts = append(filePrompts, fmt.Sprintf("Complexity is %d (Max 8 limit). Nudge agent to extract nested conditionals into separate helper functions.", res.Metrics.Complexity))
			}
			if res.Metrics.FunctionLength > 50 {
				if !hasViolations {
					totalViolations++
					hasViolations = true
				}
				filePrompts = append(filePrompts, fmt.Sprintf("Function lines is %d (Max 50 limit). Nudge agent to modularize this block into separate functional components.", res.Metrics.FunctionLength))
			}
			if res.Metrics.ArgumentCount > 4 {
				if !hasViolations {
					totalViolations++
					hasViolations = true
				}
				filePrompts = append(filePrompts, fmt.Sprintf("Parameter count is %d (Max 4 limit). Nudge agent to bundle parameters into a structured configuration object.", res.Metrics.ArgumentCount))
			}

			// Render Table Row
			compClass := ""
			if res.Metrics.Complexity > 8 {
				compClass = "text-error font-bold"
			}
			linesClass := ""
			if res.Metrics.FunctionLength > 50 {
				linesClass = "text-error font-bold"
			}
			paramsClass := ""
			if res.Metrics.ArgumentCount > 4 {
				paramsClass = "text-error font-bold"
			}

			tableRows.WriteString(fmt.Sprintf(`
				<tr>
					<td><span class="file-name">%s</span><br><span class="file-path">%s</span></td>
					<td><span class="badge badge-lang">%s</span></td>
					<td class="%s">%d</td>
					<td class="%s">%d</td>
					<td class="%s">%d</td>
					<td><span class="badge badge-success">ORCHESTRATED (L1+)</span></td>
				</tr>
			`, html.EscapeString(fileBase), html.EscapeString(res.FilePath), strings.ToUpper(res.Language),
				compClass, res.Metrics.Complexity,
				linesClass, res.Metrics.FunctionLength,
				paramsClass, res.Metrics.ArgumentCount))

			// Render Prompts Card if any
			if len(filePrompts) > 0 {
				var promptLines strings.Builder
				for _, p := range filePrompts {
					promptLines.WriteString(fmt.Sprintf("<li>%s</li>", html.EscapeString(p)))
				}
				promptCards.WriteString(fmt.Sprintf(`
					<div class="card-viol">
						<h4>📄 %s</h4>
						<ul>%s</ul>
						<div class="copy-box">
							<pre id="prompt-%s">%s</pre>
							<button onclick="copyToClipboard('prompt-%s')">Copy Prompt</button>
						</div>
					</div>
				`, html.EscapeString(fileBase), promptLines.String(),
					html.EscapeString(fileBase),
					html.EscapeString(fmt.Sprintf("Refactor %s. Violations:\n%s", fileBase, strings.Join(filePrompts, "\n"))),
					html.EscapeString(fileBase)))
			}

			// Render Exceptions Card if any
			if len(res.Exceptions) > 0 {
				var excLines strings.Builder
				for _, exc := range res.Exceptions {
					excLines.WriteString(fmt.Sprintf(`
						<div class="exc-line">
							<span class="exc-rule">%s:</span> 
							<span class="exc-val">Configured Limit %d</span> 
							<span class="exc-base">(Standard Baseline is %d)</span>
						</div>
					`, html.EscapeString(exc.RuleName), exc.ConfiguredVal, exc.BaselineVal))
				}
				exceptionCards.WriteString(fmt.Sprintf(`
					<div class="card-exc">
						<h4>📄 %s</h4>
						%s
					</div>
				`, html.EscapeString(fileBase), excLines.String()))
			}

		} else {
			blindCount++
			tableRows.WriteString(fmt.Sprintf(`
				<tr>
					<td><span class="file-name">%s</span><br><span class="file-path">%s</span></td>
					<td><span class="badge badge-lang">%s</span></td>
					<td class="text-muted">-</td>
					<td class="text-muted">-</td>
					<td class="text-muted">-</td>
					<td><span class="badge badge-warn">WORKING BLIND (L0)</span></td>
				</tr>
			`, html.EscapeString(fileBase), html.EscapeString(res.FilePath), strings.ToUpper(res.Language)))
		}
	}

	// Render prompt section visibility helper
	promptsSection := `<div class="empty-state">🎉 Zero maintainability violations detected! Your coding agents are producing highly structured, clean code.</div>`
	if totalViolations > 0 {
		promptsSection = promptCards.String()
	}

	// Render exception section visibility helper
	exceptionsSection := `<div class="empty-state">No relaxed limits detected. The codebase is strictly conforming to standard baselines.</div>`
	if totalExceptions > 0 {
		exceptionsSection = exceptionCards.String() + `
			<div class="quote-box">
				<p>💡 <em>“Looking at the exceptions AI created was a good point to start my code review.”</em></p>
				<span class="quote-author">— Birgitta Böckeler, Thoughtworks</span>
			</div>`
	}

	// Full responsive HTML structure styled beautifully
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>📡 Maintainability Sensors Scorecard</title>
	<style>
		:root {
			--bg-main: #0f172a;
			--bg-card: #1e293b;
			--text-main: #f8fafc;
			--text-muted: #94a3b8;
			--accent: #38bdf8;
			--accent-hover: #0ea5e9;
			--border: #334155;
			--success: #10b981;
			--warning: #f59e0b;
			--error: #ef4444;
		}

		* {
			box-sizing: border-box;
			margin: 0;
			padding: 0;
		}

		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
			background-color: var(--bg-main);
			color: var(--text-main);
			line-height: 1.5;
			padding: 2rem 1rem;
		}

		.container {
			max-width: 1200px;
			margin: 0 auto;
		}

		header {
			margin-bottom: 2rem;
			border-bottom: 1px solid var(--border);
			padding-bottom: 1.5rem;
		}

		h1 {
			font-size: 2.25rem;
			font-weight: 800;
			color: #ffffff;
			display: flex;
			align-items: center;
			gap: 0.75rem;
		}

		h2 {
			font-size: 1.5rem;
			font-weight: 700;
			margin-bottom: 1rem;
			color: #ffffff;
		}

		h3 {
			font-size: 1.15rem;
			font-weight: 600;
			margin-bottom: 0.5rem;
		}

		p.subtitle {
			color: var(--text-muted);
			font-size: 1.1rem;
			margin-top: 0.25rem;
		}

		/* Grid Stats */
		.stats-grid {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
			gap: 1.25rem;
			margin-bottom: 2.5rem;
		}

		.stat-card {
			background-color: var(--bg-card);
			border: 1px solid var(--border);
			border-radius: 0.75rem;
			padding: 1.25rem;
			text-align: center;
		}

		.stat-val {
			font-size: 2rem;
			font-weight: 800;
			color: #ffffff;
			margin-bottom: 0.25rem;
		}

		.stat-card.error .stat-val { color: var(--error); }
		.stat-card.warning .stat-val { color: var(--warning); }
		.stat-card.success .stat-val { color: var(--success); }

		.stat-label {
			color: var(--text-muted);
			font-size: 0.875rem;
			font-weight: 500;
			text-transform: uppercase;
			letter-spacing: 0.05em;
		}

		/* Main Table */
		.section-card {
			background-color: var(--bg-card);
			border: 1px solid var(--border);
			border-radius: 0.75rem;
			padding: 1.5rem;
			margin-bottom: 2.5rem;
			overflow-x: auto;
		}

		table {
			width: 100%%;
			border-collapse: collapse;
			text-align: left;
		}

		th, td {
			padding: 1rem;
			border-bottom: 1px solid var(--border);
		}

		th {
			font-weight: 600;
			color: var(--text-muted);
			font-size: 0.875rem;
			text-transform: uppercase;
			letter-spacing: 0.05em;
		}

		tr:last-child td {
			border-bottom: none;
		}

		.file-name {
			font-weight: 600;
			color: #ffffff;
		}

		.file-path {
			font-size: 0.75rem;
			color: var(--text-muted);
		}

		/* Badges */
		.badge {
			display: inline-block;
			padding: 0.25rem 0.6rem;
			border-radius: 9999px;
			font-size: 0.75rem;
			font-weight: 600;
			text-transform: uppercase;
		}

		.badge-lang {
			background-color: rgba(56, 189, 248, 0.15);
			color: var(--accent);
			border: 1px solid rgba(56, 189, 248, 0.3);
		}

		.badge-success {
			background-color: rgba(16, 185, 129, 0.15);
			color: var(--success);
			border: 1px solid rgba(16, 185, 129, 0.3);
		}

		.badge-warn {
			background-color: rgba(245, 158, 11, 0.15);
			color: var(--warning);
			border: 1px solid rgba(245, 158, 11, 0.3);
		}

		/* Split Layout */
		.split-layout {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(450px, 1fr));
			gap: 2rem;
		}

		.card-viol, .card-exc {
			background-color: #151f32;
			border: 1px solid var(--border);
			border-radius: 0.5rem;
			padding: 1.25rem;
			margin-bottom: 1.25rem;
		}

		.card-viol h4, .card-exc h4 {
			color: #ffffff;
			margin-bottom: 0.75rem;
			font-size: 1rem;
		}

		.card-viol ul {
			list-style-position: inside;
			margin-bottom: 1rem;
			color: var(--text-muted);
			font-size: 0.9rem;
		}

		.card-viol li {
			margin-bottom: 0.4rem;
		}

		.copy-box {
			background-color: #0b0f19;
			border: 1px solid var(--border);
			border-radius: 0.375rem;
			padding: 0.75rem;
			position: relative;
		}

		.copy-box pre {
			font-family: "Courier New", Courier, monospace;
			font-size: 0.85rem;
			color: #38bdf8;
			white-space: pre-wrap;
			word-break: break-all;
			padding-bottom: 2rem;
		}

		.copy-box button {
			position: absolute;
			bottom: 0.5rem;
			right: 0.5rem;
			background-color: var(--accent);
			color: #000000;
			border: none;
			padding: 0.35rem 0.75rem;
			border-radius: 0.25rem;
			font-size: 0.75rem;
			font-weight: 700;
			cursor: pointer;
			transition: background-color 0.15s ease;
		}

		.copy-box button:hover {
			background-color: var(--accent-hover);
		}

		/* Exceptions style */
		.exc-line {
			display: flex;
			align-items: center;
			justify-content: space-between;
			border-bottom: 1px solid rgba(255, 255, 255, 0.05);
			padding: 0.5rem 0;
			font-size: 0.9rem;
		}

		.exc-line:last-child {
			border-bottom: none;
		}

		.exc-rule {
			font-weight: 600;
			color: var(--text-main);
		}

		.exc-val {
			color: var(--warning);
			font-weight: 600;
		}

		.exc-base {
			color: var(--text-muted);
			font-size: 0.8rem;
		}

		.quote-box {
			background-color: rgba(245, 158, 11, 0.05);
			border-left: 4px solid var(--warning);
			padding: 1rem;
			border-radius: 0 0.5rem 0.5rem 0;
			margin-top: 1.5rem;
		}

		.quote-box p {
			color: var(--text-main);
			font-style: italic;
			font-size: 0.95rem;
		}

		.quote-author {
			display: block;
			text-align: right;
			font-size: 0.8rem;
			color: var(--text-muted);
			margin-top: 0.25rem;
		}

		.empty-state {
			text-align: center;
			color: var(--text-muted);
			padding: 2rem;
			font-size: 0.95rem;
		}

		.text-error { color: var(--error) !important; }
		.text-muted { color: var(--text-muted); }
		.font-bold { font-weight: bold; }

		@media (max-width: 768px) {
			.split-layout {
				grid-template-columns: 1fr;
			}
		}
	</style>
</head>
<body>
	<div class="container">
		<header>
			<h1>📡 Maintainability Sensors Scorecard</h1>
			<p class="subtitle">Continuous engineering governance & metrics loop for AI coding agents</p>
		</header>

		<!-- Statistics Summary Grid -->
		<div class="stats-grid">
			<div class="stat-card">
				<div class="stat-val">%d</div>
				<div class="stat-label">Total Files</div>
			</div>
			<div class="stat-card success">
				<div class="stat-val">%d</div>
				<div class="stat-label">Orchestrated</div>
			</div>
			<div class="stat-card warning">
				<div class="stat-val">%d</div>
				<div class="stat-label">Level 0 (Blind)</div>
			</div>
			<div class="stat-card error">
				<div class="stat-val">%d</div>
				<div class="stat-label">Violations</div>
			</div>
			<div class="stat-card warning">
				<div class="stat-val">%d</div>
				<div class="stat-label">AI Exceptions</div>
			</div>
		</div>

		<!-- Main Scorecard Table -->
		<section>
			<h2>📊 Core File Health Scorecard</h2>
			<div class="section-card">
				<table>
					<thead>
						<tr>
							<th>Scanned File</th>
							<th>Language</th>
							<th>Complexity</th>
							<th>Func Lines</th>
							<th>Max Params</th>
							<th>Sensor Status</th>
						</tr>
					</thead>
					<tbody>
						%s
					</tbody>
				</table>
			</div>
		</section>

		<!-- Dynamic Double Column Details -->
		<div class="split-layout">
			<!-- AI Prompts Column -->
			<section>
				<h2>🤖 AI Agent Self-Correction Prompts</h2>
				<p class="subtitle" style="margin-bottom: 1rem; font-size: 0.9rem;">Copy-paste these instructions directly to your AI agent to trigger auto-refactoring:</p>
				%s
			</section>

			<!-- Exceptions Column -->
			<section>
				<h2>🛠️ Exceptions Created by AI (Relaxed Limits)</h2>
				<p class="subtitle" style="margin-bottom: 1rem; font-size: 0.9rem;">Manually review relaxed constraints created by agents to prevent architectural decay:</p>
				%s
			</section>
		</div>
	</div>

	<script>
		function copyToClipboard(id) {
			const element = document.getElementById(id);
			const range = document.createRange();
			range.selectNode(element);
			window.getSelection().removeAllRanges();
			window.getSelection().addRange(range);
			try {
				document.execCommand('copy');
				const btn = element.nextElementSibling;
				const originalText = btn.innerText;
				btn.innerText = 'Copied!';
				btn.style.backgroundColor = '#10b981';
				btn.style.color = '#ffffff';
				setTimeout(() => {
					btn.innerText = originalText;
					btn.style.backgroundColor = 'var(--accent)';
					btn.style.color = '#000000';
				}, 1500);
			} catch (err) {
				alert('Failed to copy to clipboard.');
			}
			window.getSelection().removeAllRanges();
		}
	</script>
</body>
</html>`,
		totalFiles, orchestratedCount, blindCount, totalViolations, totalExceptions,
		tableRows.String(), promptsSection, exceptionsSection)
}
