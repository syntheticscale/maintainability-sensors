package sensors

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/pelletier/go-toml/v2"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"gopkg.in/yaml.v3"
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

func extractVal(vv interface{}, vals *[]int) {
	switch actual := vv.(type) {
	case float64:
		*vals = append(*vals, int(actual))
	case int:
		*vals = append(*vals, actual)
	case int64:
		*vals = append(*vals, int(actual))
	case map[string]interface{}:
		if maxVal, ok := actual["max"]; ok {
			extractVal(maxVal, vals)
		} else if maxVal, ok := actual["Max"]; ok {
			extractVal(maxVal, vals)
		}
	case map[interface{}]interface{}:
		if maxVal, ok := actual["max"]; ok {
			extractVal(maxVal, vals)
		} else if maxVal, ok := actual["Max"]; ok {
			extractVal(maxVal, vals)
		}
	case []interface{}:
		for _, item := range actual {
			extractVal(item, vals)
		}
	}
}

// findAllConfigVals searches content for key and extracts all associated integer values.
// If ext is ".json" it uses a recursive JSON walker; if ".js" or ".mjs" it uses a regex parser; otherwise it uses YAML/TOML/INI parsers.
func findAllConfigVals(content string, key string, ext string) []int {
	if ext == ".json" {
		return findAllConfigValsJSON(content, key)
	}
	if ext == ".js" || ext == ".mjs" {
		return findAllConfigValsJS(content, key)
	}
	if ext == ".toml" {
		return findAllConfigValsTOML(content, key)
	}

	return findAllConfigValsYAML(content, key)
}

func findAllConfigValsJS(content string, key string) []int {
	var vals []int

	parser := sitter.NewParser()
	parser.SetLanguage(typescript.GetLanguage())

	tree, _ := parser.ParseCtx(context.Background(), nil, []byte(content))
	if tree == nil {
		return vals
	}
	rootNode := tree.RootNode()

	var extractVal func(*sitter.Node) []int
	extractVal = func(n *sitter.Node) []int {
		var extracted []int
		switch n.Type() {
		case "number":
			if val, err := strconv.Atoi(string(content[n.StartByte():n.EndByte()])); err == nil {
				extracted = append(extracted, val)
			}
		case "array":
			for i := 0; i < int(n.NamedChildCount()); i++ {
				extracted = append(extracted, extractVal(n.NamedChild(i))...)
			}
		case "object":
			for i := 0; i < int(n.NamedChildCount()); i++ {
				child := n.NamedChild(i)
				if child.Type() == "pair" {
					kNode := child.ChildByFieldName("key")
					if kNode != nil {
						kStr := string(content[kNode.StartByte():kNode.EndByte()])
						if kStr == "max" || kStr == "\"max\"" || kStr == "'max'" || kStr == "Max" || kStr == "\"Max\"" || kStr == "'Max'" {
							vNode := child.ChildByFieldName("value")
							if vNode != nil {
								extracted = append(extracted, extractVal(vNode)...)
							}
						}
					}
				}
			}
		}
		return extracted
	}

	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "pair" {
			kNode := n.ChildByFieldName("key")
			if kNode != nil {
				kStr := string(content[kNode.StartByte():kNode.EndByte()])
				if kStr == key || kStr == "\""+key+"\"" || kStr == "'"+key+"'" {
					vNode := n.ChildByFieldName("value")
					if vNode != nil {
						vals = append(vals, extractVal(vNode)...)
					}
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(rootNode)

	sort.Ints(vals)
	return vals
}

func findAllConfigValsYAML(content string, key string) []int {
	var vals []int
	var walk func(interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			for k, vv := range val {
				if k == key || (len(k) > len(key) && k[len(k)-len(key)-1:] == "/"+key) {
					extractVal(vv, &vals)
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
	yaml.Unmarshal([]byte(content), &data)
	walk(data)

	if len(vals) == 0 {
		// Fallback for ini style like pylintrc
		pattern := fmt.Sprintf(`(?m)^%s\s*(?:=|:)\s*(\d+)$`, regexp.QuoteMeta(key))
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				if val, err := strconv.Atoi(match[1]); err == nil {
					vals = append(vals, val)
				}
			}
		}
	}

	sort.Ints(vals)
	return vals
}

func findAllConfigValsTOML(content string, key string) []int {
	var vals []int
	var walk func(interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			for k, vv := range val {
				if k == key || (len(k) > len(key) && k[len(k)-len(key)-1:] == "/"+key) {
					extractVal(vv, &vals)
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
	toml.Unmarshal([]byte(content), &data)
	walk(data)

	sort.Ints(vals)
	return vals
}

func findAllConfigValsJSON(content string, key string) []int {
	var vals []int
	var walk func(interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			for k, vv := range val {
				if k == key || (len(k) > len(key) && k[len(k)-len(key)-1:] == "/"+key) {
					extractVal(vv, &vals)
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
