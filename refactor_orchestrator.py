import re

with open("sensors/orchestrator.go", "r") as f:
    content = f.read()

# 1. Update OrchestratedScanBatch
content = re.sub(
    r'toolMetrics := make\(map\[string\]MaintainabilityMetrics\)',
    r'toolMetrics := make(map[string][]Violation)',
    content
)

content = re.sub(
    r'metricsMap\[p\] = outMetrics',
    r'''metrics := metricsMap[p]
						for _, v := range outMetrics {
							switch v.RuleName {
							case "Complexity":
								if v.Value > metrics.Complexity {
									metrics.Complexity = v.Value
								}
							case "FunctionLength":
								if v.Value > metrics.FunctionLength {
									metrics.FunctionLength = v.Value
								}
							case "ArgumentCount":
								if v.Value > metrics.ArgumentCount {
									metrics.ArgumentCount = v.Value
								}
							}
						}
						metricsMap[p] = metrics''',
    content
)

# 2. Update all plugins in orchestrator.go to return []Violation
plugins = ["ESLintPlugin", "PyLintPlugin", "RuboCopPlugin", "RuffPlugin", "BiomePlugin", "StandardRBPlugin"]

for plugin in plugins:
    content = re.sub(
        rf'func \(p {plugin}\) Analyze\(filePaths \[\]string\) \(map\[string\]MaintainabilityMetrics, error\) {{',
        rf'func (p {plugin}) Analyze(filePaths []string) (map[string][]Violation, error) {{',
        content
    )

with open("sensors/orchestrator.go", "w") as f:
    f.write(content)
