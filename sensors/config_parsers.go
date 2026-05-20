package sensors

import (
	"encoding/json"
	"fmt"
	"regexp"
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
// If ext is ".json" it uses a recursive JSON walker; otherwise it uses line-oriented regex.
func findAllConfigVals(content string, key string, ext string) []int {
	if ext == ".json" {
		return findAllConfigValsJSON(content, key)
	}

	// Line-oriented approach for INI/YAML-style files
	var vals []int
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		if !strings.Contains(line, key) {
			continue
		}
		// Require the key to be a whole word and not followed immediately by a hyphen
		pattern := `\b` + regexp.QuoteMeta(key) + `\b[^-\d]*?(\d+)`
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			var val int
			if _, err := fmt.Sscanf(matches[1], "%d", &val); err == nil {
				vals = append(vals, val)
			}
		}
	}
	return vals
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
