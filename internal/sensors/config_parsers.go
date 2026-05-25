package sensors

//nolint // maintainability: highly cohesive logic

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

func extractMapVal(actual map[string]interface{}, vals *[]int) {
	if maxVal, ok := actual["max"]; ok {
		extractVal(maxVal, vals)
	} else if maxVal, ok := actual["Max"]; ok {
		extractVal(maxVal, vals)
	}
}

func extractInterfaceMapVal(actual map[interface{}]interface{}, vals *[]int) {
	if maxVal, ok := actual["max"]; ok {
		extractVal(maxVal, vals)
	} else if maxVal, ok := actual["Max"]; ok {
		extractVal(maxVal, vals)
	}
}

func extractSliceVal(actual []interface{}, vals *[]int) {
	for _, item := range actual {
		extractVal(item, vals)
	}
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic for types. Splitting this hurts readability.
func extractVal(vv interface{}, vals *[]int) {
	switch actual := vv.(type) {
	case float64:
		*vals = append(*vals, int(actual))
	case int:
		*vals = append(*vals, actual)
	case int64:
		*vals = append(*vals, int(actual))
	case map[string]interface{}:
		extractMapVal(actual, vals)
	case map[interface{}]interface{}:
		extractInterfaceMapVal(actual, vals)
	case []interface{}:
		extractSliceVal(actual, vals)
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

func jsIsMaxKey(kStr string) bool {
	return kStr == "max" || kStr == "\"max\"" || kStr == "'max'" || kStr == "Max" || kStr == "\"Max\"" || kStr == "'Max'"
}

func jsIsMatchingKey(kStr, key string) bool {
	return kStr == key || kStr == "\""+key+"\"" || kStr == "'"+key+"'"
}

func jsExtractNumber(content string, n *sitter.Node) []int {
	if val, err := strconv.Atoi(string(content[n.StartByte():n.EndByte()])); err == nil {
		return []int{val}
	}
	return nil
}

func jsExtractArray(content string, n *sitter.Node) []int {
	var extracted []int
	for i := 0; i < int(n.NamedChildCount()); i++ {
		extracted = append(extracted, jsExtractVal(content, n.NamedChild(i))...)
	}
	return extracted
}

func jsProcessPair(content string, child *sitter.Node) []int {
	if child.Type() != "pair" {
		return nil
	}
	kNode := child.ChildByFieldName("key")
	if kNode == nil {
		return nil
	}
	kStr := string(content[kNode.StartByte():kNode.EndByte()])
	if jsIsMaxKey(kStr) {
		if vNode := child.ChildByFieldName("value"); vNode != nil {
			return jsExtractVal(content, vNode)
		}
	}
	return nil
}

func jsExtractObject(content string, n *sitter.Node) []int {
	var extracted []int
	for i := 0; i < int(n.NamedChildCount()); i++ {
		extracted = append(extracted, jsProcessPair(content, n.NamedChild(i))...)
	}
	return extracted
}

func jsExtractVal(content string, n *sitter.Node) []int {
	switch n.Type() {
	case "number":
		return jsExtractNumber(content, n)
	case "array":
		return jsExtractArray(content, n)
	case "object":
		return jsExtractObject(content, n)
	}
	return nil
}

func processJSPair(content string, key string, n *sitter.Node, vals *[]int) {
	kNode := n.ChildByFieldName("key")
	if kNode == nil {
		return
	}
	kStr := string(content[kNode.StartByte():kNode.EndByte()])
	if !jsIsMatchingKey(kStr, key) {
		return
	}
	vNode := n.ChildByFieldName("value")
	if vNode != nil {
		*vals = append(*vals, jsExtractVal(content, vNode)...)
	}
}

func jsWalk(content string, key string, n *sitter.Node, vals *[]int) {
	if n.Type() == "pair" {
		processJSPair(content, key, n, vals)
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		jsWalk(content, key, n.NamedChild(i), vals)
	}
}

func findAllConfigValsJS(content string, key string) []int {
	var vals []int

	parser := sitter.NewParser()
	parser.SetLanguage(typescript.GetLanguage())

	tree, _ := parser.ParseCtx(context.Background(), nil, []byte(content))
	if tree == nil {
		return vals
	}

	jsWalk(content, key, tree.RootNode(), &vals)

	sort.Ints(vals)
	return vals
}

func mapHasMatchingKey(k, key string) bool {
	return k == key || (len(k) > len(key) && k[len(k)-len(key)-1:] == "/"+key)
}

func genericWalk(v interface{}, key string, vals *[]int) {
	switch val := v.(type) {
	case map[string]interface{}:
		walkMapStringInterface(val, key, vals)
	case []interface{}:
		walkSliceInterface(val, key, vals)
	}
}

//nolint:gocognit,cyclop // walking interfaces requires checks
func walkMapStringInterface(val map[string]interface{}, key string, vals *[]int) {
	for k, vv := range val {
		if mapHasMatchingKey(k, key) {
			extractVal(vv, vals)
		}
		genericWalk(vv, key, vals)
	}
}

func walkSliceInterface(val []interface{}, key string, vals *[]int) {
	for _, item := range val {
		genericWalk(item, key, vals)
	}
}

func findAllConfigValsYAML(content string, key string) []int {
	var vals []int
	var data interface{}
	yaml.Unmarshal([]byte(content), &data)
	genericWalk(data, key, &vals)

	if len(vals) == 0 {
		vals = append(vals, extractFallbackIniVals(content, key)...)
	}

	sort.Ints(vals)
	return vals
}

func extractFallbackIniVals(content string, key string) []int {
	var vals []int
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
	return vals
}

func findAllConfigValsTOML(content string, key string) []int {
	var vals []int
	var data interface{}
	toml.Unmarshal([]byte(content), &data)
	genericWalk(data, key, &vals)

	sort.Ints(vals)
	return vals
}

func findAllConfigValsJSON(content string, key string) []int {
	var vals []int
	var data interface{}
	if err := json.Unmarshal([]byte(content), &data); err == nil {
		genericWalk(data, key, &vals)
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

