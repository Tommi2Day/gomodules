package common

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	const fallback = "NotFound"
	t.Run("Test Strin Env", func(t *testing.T) {
		key := "TESTKEY"

		expected := "Test"
		expectedType := "string"
		_ = os.Setenv(key, expected)
		actual := GetStringEnv(key, fallback)
		assert.NotEmpty(t, actual, "Value Empty")
		assert.Equal(t, expected, actual, "value not expected")
		assert.IsTypef(t, expectedType, actual, "Type mismatch, expected:%s, actual:%s", expected, actual)
	})
	t.Run("Test nonexisting Env", func(t *testing.T) {
		actual := GetEnv("NoKey", fallback)
		assert.NotEmpty(t, actual, "Value Empty")
		assert.Equal(t, fallback, actual, "value not expected")
	})
	t.Run("Test int Env", func(t *testing.T) {
		key := "INTKEY"
		fallback := 0
		expected := 123
		_ = os.Setenv(key, fmt.Sprintf("%d", expected))
		actual := GetIntEnv(key, fallback)
		assert.NotEmpty(t, actual, "Value Empty")
		assert.Equal(t, expected, actual, "value not expected")
		assert.IsTypef(t, expected, actual, "Type mismatch")
	})

	t.Run("Test Float Env", func(t *testing.T) {
		key := "FLOATKEY"
		fallback := 0.0
		expected := 123.321
		_ = os.Setenv(key, fmt.Sprintf("%f", expected))
		actual := GetFloatEnv(key, fallback)
		assert.NotEmpty(t, actual, "Value Empty")
		assert.Equal(t, expected, actual, "value not expected")
		assert.IsType(t, expected, actual, "Type mismatch")
	})
	t.Run("Test Bool Env", func(t *testing.T) {
		const expected = true
		key := "BOOLKEY"
		_ = os.Setenv(key, fmt.Sprintf("%v", expected))
		actual := GetBoolEnv(key, false)
		assert.NotEmpty(t, actual, "Value Empty")
		assert.Equal(t, expected, actual, "value not expected")
		assert.IsTypef(t, expected, actual, "Type mismatch")
	})
}

func TestRemoveSpace(t *testing.T) {
	d := `
# abc

    def

`
	actual := RemoveSpace(d)
	expected := "#abcdef"
	assert.Equal(t, expected, actual, "Not all withespace removed")
}

func TestCheckSkip(t *testing.T) {
	t.Run("Check Skip logic", func(t *testing.T) {
		type testTableType struct {
			name  string
			input string
			skip  bool
		}
		for _, testconfig := range []testTableType{
			{
				name:  "Test comment",
				input: "# comment",
				skip:  true,
			},
			{
				name:  "Empty line 1",
				input: "",
				skip:  true,
			},
			{
				name:  "Empty line 2",
				input: "          \t",
				skip:  true,
			},
			{
				name:  "Test normal line",
				input: "test test",
				skip:  false,
			},
			{
				name:  "Test comment after code",
				input: " not to skip # comment",
				skip:  false,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				actual := CheckSkip(testconfig.input)
				assert.Equal(t, testconfig.skip, actual, "unexpected answer")
			})
		}
	})
}
func TestExecuteOsCommand(t *testing.T) {
	var cmdArg []string
	myOs := runtime.GOOS
	switch myOs {
	case osWin:
		cmdArg = []string{"cmd.exe", "/c", "dir"}
	default:
		cmdArg = []string{"/bin/ls"}
	}

	stdout, stderr, err := ExecuteOsCommand(cmdArg, nil)
	assert.NoErrorf(t, err, "Command got Error: %v", err)
	assert.Emptyf(t, stderr, "StdErr not empty")
	assert.NotEmpty(t, stdout, "Output is empty")
}

func TestCommandExists(t *testing.T) {
	t.Run("Test Command Exists", func(t *testing.T) {
		// Test with existing command
		c := "ls"
		if runtime.GOOS == osWin {
			c = "cmd.exe"
		}
		actual := CommandExists(c)
		assert.Truef(t, actual, "Command %s not found", c)

		// Test with non existing command
		actual = CommandExists("nonexistingcommand")
		assert.False(t, actual, "nonexisting Command found")
	})
}
func TestInArray(t *testing.T) {
	type testTableType struct {
		name     string
		needle   interface{}
		haystack []interface{}
		result   bool
		index    int
	}
	for _, testconfig := range []testTableType{
		{
			name:     "Test String",
			needle:   "needle",
			haystack: []interface{}{"needle", "haystack"},
			result:   true,
			index:    0,
		},
		{
			name:     "Test failed String",
			needle:   "no needle",
			haystack: []interface{}{"needle", "haystack"},
			result:   false,
			index:    -1,
		},
		{
			name:     "Test int",
			needle:   123,
			haystack: []interface{}{1, 2, 3, 123},
			result:   true,
			index:    3,
		},
		{
			name:     "Test failed int",
			needle:   123,
			haystack: []interface{}{1, 2, 3},
			result:   false,
			index:    -1,
		},
	} {
		t.Run(testconfig.name, func(t *testing.T) {
			actual, index := InArray(testconfig.needle, testconfig.haystack)
			assert.Equal(t, testconfig.result, actual, "unexpected answer")
			assert.Equal(t, testconfig.index, index, "unexpected index")
		})
	}
}

func TestFormatUnixtsString(t *testing.T) {
	type testTableType struct {
		name     string
		input    string
		layout   string
		expected string
	}
	ts := time.Now().Unix()

	for _, testconfig := range []testTableType{
		{
			name:     "FormatUnixtsStringRFC3339",
			input:    fmt.Sprintf("%d", ts),
			layout:   time.RFC822,
			expected: time.Unix(ts, 0).Format(time.RFC822),
		},
		{
			name:     "FormatUnixtsStringRFC822",
			input:    fmt.Sprintf("%d", ts),
			layout:   time.RFC822,
			expected: time.Unix(ts, 0).Format(time.RFC822),
		},
		{
			name:     "FormatUnixtsStringRFCNotNumeric",
			input:    fmt.Sprintf("%dxxx", ts),
			layout:   time.RFC822,
			expected: fmt.Sprintf("%dxxx", ts),
		},
	} {
		t.Run(testconfig.name, func(t *testing.T) {
			actual := FormatUnixtsString(testconfig.input, testconfig.layout)
			assert.Equal(t, testconfig.expected, actual, "unexpected answer")
		})
	}
}
func TestReverseMap(t *testing.T) {
	input := map[string]int{
		"one": 1,
		"two": 2,
	}
	expected := map[int]string{
		1: "one",
		2: "two",
	}
	actual := ReverseMap(input)
	assert.Equal(t, expected, actual, "Reverse Map not as expected")
}
