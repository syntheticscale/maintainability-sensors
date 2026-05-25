package tests

//nolint // maintainability: highly cohesive test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func TestParseTypeScriptTreeSitter(t *testing.T) {
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.ts")
	tsContent := `
function simpleFunc(a: string, b: number) {
	console.log(a, b);
}

class TestClass {
	complexMethod(x: number, y: number, z: number) {
		let result = 0;
		if (x > 0) {
			result += x;
		} else if (x < 0) {
			result -= x;
		}

		for (let i = 0; i < y; i++) {
			result += i;
		}

		let j = 0;
		while (j < z) {
			result += j;
			j++;
		}

		switch (z) {
			case 1:
				result += 1;
				break;
			case 2:
				result += 2;
				break;
		}

		try {
			throw new Error("test");
		} catch (e) {
			result = 0;
		}

		return result;
	}
}

const arrowFunc = (a: string) => {
	return a;
};
`
	err := os.WriteFile(testFile, []byte(tsContent), 0644)
	require.NoError(t, err)

	violations, err := sensors.ParseTypeScriptTreeSitter(sensors.FileContext{Path: testFile})
	require.NoError(t, err)

	for _, v := range violations {
		t.Logf("Violation: %+v\n", v)
	}

	// Check simpleFunc
	assertViolation(t, violations, "FunctionLength", 3, 2, 4)
	assertViolation(t, violations, "ArgumentCount", 2, 2, 4)
	assertViolation(t, violations, "Complexity", 1, 2, 4)

	// Check complexMethod
	assertViolation(t, violations, "FunctionLength", 35, 7, 41)
	assertViolation(t, violations, "ArgumentCount", 3, 7, 41)
	assertViolation(t, violations, "Complexity", 8, 7, 41) // base=1, if=2, for=1, while=1, case=2, catch=1

	// Check arrowFunc
	assertViolation(t, violations, "FunctionLength", 3, 44, 46)
	assertViolation(t, violations, "ArgumentCount", 1, 44, 46)
	assertViolation(t, violations, "Complexity", 1, 44, 46)
}

func assertViolation(t *testing.T, violations []sensors.Violation, ruleName string, expectedValue int, startLine int, endLine int) {
	found := false
	for _, v := range violations {
		if v.RuleName == ruleName && v.Value == expectedValue && v.StartLine == startLine && v.EndLine == endLine {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected violation %s with value %d from line %d to %d not found", ruleName, expectedValue, startLine, endLine)
}
