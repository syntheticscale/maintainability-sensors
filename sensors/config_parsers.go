package sensors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ParserRule maps a human-readable rule name to the config key(s) used to look it up.
type ParserRule struct {
	RuleName string
	Keys     []string
	Baseline int
}

// ConfigParser extracts threshold values from a config file for a given language.
type ConfigParser interface {
	// Name returns the parser identifier (for diagnostics).
	Name() string
	// Rules returns the ordered list of rules this parser knows about.
	Rules() []ParserRule
	// Anchors returns the list of file names to search for when locating a config file.
	Anchors() []string
}

// findAllConfigVals searches content for key and extracts all associated integer values.
// If ext is ".json" it uses a recursive JSON walker; if ".js" or ".mjs" it uses a regex parser; otherwise it uses a custom YAML/INI subset parser.
func findAllConfigVals(content string, key string, ext string) []int {
	if ext == ".json" {
		return findAllConfigValsJSON(content, key)
	}
	if ext == ".js" || ext == ".mjs" {
		return findAllConfigValsJS(content, key)
	}

	// Line-oriented approach using custom YAML/INI subset parser
	parsed := parseYAMLSubset(content)
	if val, ok := parsed[key]; ok {
		switch v := val.(type) {
		case int:
			return []int{v}
		case []int:
			return v
		}
	}
	return nil
}

func findAllConfigValsJS(content string, key string) []int {
	var vals []int

	// Matches: "key": ["error", 20] or key: ["warn", 15] or "key": 5 or key: 5
	// Groups: 1 -> array value, 2 -> primitive value
	safeKey := regexp.QuoteMeta(key)
	pattern := fmt.Sprintf(`(?:["']%s["']|\b%s\b)\s*:\s*(?:\[\s*["'][^"']+["']\s*,\s*(\d+)\s*\]|(\d+))`, safeKey, safeKey)
	re := regexp.MustCompile(pattern)

	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		var numStr string
		if len(match) > 1 && match[1] != "" {
			numStr = match[1]
		} else if len(match) > 2 && match[2] != "" {
			numStr = match[2]
		}

		if numStr != "" {
			if val, err := strconv.Atoi(numStr); err == nil {
				vals = append(vals, val)
			}
		}
	}

	sort.Ints(vals)
	return vals
}

type indentNode struct {
	indent int
	key    string
}

func parseYAMLSubset(content string) map[string]interface{} {
	res := make(map[string]interface{})
	lines := strings.Split(content, "\n")
	var stack []indentNode

	for _, line := range lines {
		if isIgnoredYAMLLine(line) {
			continue
		}

		indent := getYAMLIndent(line)
		line = strings.TrimSpace(line)

		key, valStr, ok := splitYAMLKeyValue(line)
		if !ok {
			continue
		}

		stack = popYAMLStack(stack, indent)

		var val int
		if _, err := fmt.Sscanf(valStr, "%d", &val); err == nil {
			updateYAMLResult(res, stack, key, val)
		} else if valStr == "" {
			stack = append(stack, indentNode{indent: indent, key: key})
		}
	}
	return res
}

func isIgnoredYAMLLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "[")
}

func getYAMLIndent(line string) int {
	indent := 0
	for _, ch := range line {
		if ch == ' ' {
			indent++
		} else {
			break
		}
	}
	return indent
}

func splitYAMLKeyValue(line string) (string, string, bool) {
	idxCol := strings.Index(line, ":")
	idxEq := strings.Index(line, "=")

	var sepIdx int
	if idxCol != -1 && idxEq != -1 {
		if idxCol < idxEq {
			sepIdx = idxCol
		} else {
			sepIdx = idxEq
		}
	} else if idxCol != -1 {
		sepIdx = idxCol
	} else if idxEq != -1 {
		sepIdx = idxEq
	} else {
		return "", "", false
	}

	key := strings.TrimSpace(line[:sepIdx])
	valStr := strings.TrimSpace(line[sepIdx+1:])
	valStr = strings.Trim(valStr, "\"'")
	return key, valStr, true
}

func popYAMLStack(stack []indentNode, indent int) []indentNode {
	for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
		stack = stack[:len(stack)-1]
	}
	return stack
}

func updateYAMLResult(res map[string]interface{}, stack []indentNode, key string, val int) {
	keysToStore := []string{key}
	if len(stack) > 0 {
		parent := stack[len(stack)-1].key
		keysToStore = append(keysToStore, parent)
		if strings.Contains(parent, "/") {
			parts := strings.Split(parent, "/")
			keysToStore = append(keysToStore, parts[len(parts)-1])
		}
	}

	for _, k := range keysToStore {
		if existing, ok := res[k]; ok {
			if slice, ok := existing.([]int); ok {
				res[k] = append(slice, val)
			} else if single, ok := existing.(int); ok {
				res[k] = []int{single, val}
			}
		} else {
			res[k] = val
		}
	}
}

func findAllConfigValsJSON(content string, key string) []int {
	var vals []int
	var walk func(interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			for k, vv := range val {
				if k == key {
					switch actual := vv.(type) {
					case float64:
						vals = append(vals, int(actual))
					case []interface{}:
						for _, item := range actual {
							if f, ok := item.(float64); ok {
								vals = append(vals, int(f))
							}
						}
					}
				}
				walk(vv)
			}
		case []interface{}:
			for _, item := range val {
				walk(item)
			}
		}
	}

	var data interface{}
	if err := json.Unmarshal([]byte(content), &data); err == nil {
		walk(data)
	}
	sort.Ints(vals)
	return vals
}

func maxOf(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}
