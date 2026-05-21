import re

with open('sensors/orchestrator.go', 'r') as f:
    content = f.read()

plugins = [
    ("ESLint", "runESLintBatch", "eslint"),
    ("PyLint", "runPyLintBatch", "pylint"),
    ("RuboCop", "runRuboCopBatch", "rubocop"),
    ("Ruff", "runRuffBatch", "ruff"),
    ("Biome", "runBiomeBatch", "biome"),
    ("StandardRB", "runStandardRBBatch", "standardrb")
]

for plugin_prefix, func_name, name_str in plugins:
    # Find the function definition
    pattern = r'func ' + func_name + r'\((.*?)\) \((.*?)\) \{'
    replacement = (
        f'type {plugin_prefix}Plugin struct{{}}\n\n'
        f'func (p {plugin_prefix}Plugin) Name() string {{\n\treturn "{name_str}"\n}}\n\n'
        f'func (p {plugin_prefix}Plugin) Analyze(\\1) (\\2) {{'
    )
    content = re.sub(pattern, replacement, content, count=1)
    
    # Add back the original function as a wrapper at the end of the file
    wrapper = f'\n\nfunc {func_name}(filePaths []string) (map[string]MaintainabilityMetrics, error) {{\n\treturn {plugin_prefix}Plugin{{}}.Analyze(filePaths)\n}}\n'
    content += wrapper

with open('sensors/orchestrator.go', 'w') as f:
    f.write(content)
