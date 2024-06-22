package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConversion(t *testing.T) {
	t.Run("Check Type Conversion", func(t *testing.T) {
		type testTableType struct {
			name    string
			input   string
			success bool
			f       func(input string) error
		}
		for _, testconfig := range []testTableType{
			{
				name:    "Test IntVal",
				input:   "123",
				success: true,
				f:       testGetIntVal,
			},
			{
				name:    "Test IntVal to big",
				input:   "1234567890123456789",
				success: false,
				f:       testGetIntVal,
			},
			{
				name:    "Test IntVal err",
				input:   "a123",
				success: false,
				f:       testGetIntVal,
			},
			{
				name:    "Test Int64Val",
				input:   "-1234567890123456789",
				success: true,
				f:       testGetInt64Val,
			},
			{
				name:    "Test Int64Val to big",
				input:   "12345678901234567890", // max int64; 9223372036854775807
				success: false,
				f:       testGetInt64Val,
			},
			{
				name:    "Test BoolVal",
				input:   "true",
				success: true,
				f:       testGetBoolVal,
			},
			{
				name:    "Test BoolVal err",
				input:   "xyz",
				success: false,
				f:       testGetBoolVal,
			},
			{
				name:    "Test Float64Val",
				input:   "123456789012345.67890",
				success: true,
				f:       testGetFloatVal,
			},
			{
				name:    "Test Float64Val no dot",
				input:   "123456789012345",
				success: true,
				f:       testGetFloatVal,
			},
			{
				name:    "Test Float64Val comma",
				input:   "123456789012345,67890",
				success: false,
				f:       testGetFloatVal,
			},
			{
				name:    "Test HexInt",
				input:   "00FF00",
				success: true,
				f:       testGetHexInt64Val,
			},
			{
				name:    "Test HexInt with 0x",
				input:   "0x1234567890abcdef",
				success: true,
				f:       testGetHexInt64Val,
			},
			{
				name:    "Test HexInt wrong char",
				input:   "abcxxdqf",
				success: false,
				f:       testGetHexInt64Val,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				err := testconfig.f(testconfig.input)
				if testconfig.success {
					assert.NoErrorf(t, err, "unexpected error %s", err)
				} else {
					assert.Error(t, err, "Expected error not set")
				}
			})
		}
	})
}
func testGetIntVal(input string) error {
	_, err := GetIntVal(input)
	return err
}
func testGetInt64Val(input string) error {
	_, err := GetInt64Val(input)
	return err
}
func testGetFloatVal(input string) error {
	_, err := GetFloatVal(input)
	return err
}
func testGetBoolVal(input string) error {
	_, err := GetBoolVal(input)
	return err
}

func testGetHexInt64Val(input string) error {
	_, err := GetHexInt64Val(input)
	return err
}
func TestIsNil(t *testing.T) {
	t.Run("Test IsNil", func(t *testing.T) {
		assert.True(t, IsNil(nil))
		assert.False(t, IsNil(1))
		assert.False(t, IsNil("1"))
		assert.False(t, IsNil([]string{}))
		assert.False(t, IsNil(map[string]string{}))
		assert.False(t, IsNil(struct{}{}))
	})
}

func TestCheckType(t *testing.T) {
	t.Run("Test CheckType", func(t *testing.T) {
		type testTableType struct {
			name         string
			inputType    string
			expectedType string
			inputValue   interface{}
			success      bool
		}
		for _, testconfig := range []testTableType{
			{
				name:         "Test IntVal",
				inputType:    "int",
				expectedType: "int",
				inputValue:   123,
				success:      true,
			},
			{
				name:         "Test IntVal as string",
				inputType:    "int",
				expectedType: "string",
				inputValue:   "123",
				success:      false,
			},
			{
				name:         "Test IntVal as nil",
				inputType:    "int",
				expectedType: "<nil>",
				inputValue:   nil,
				success:      false,
			},
			{
				name:         "Test StringMapVal",
				inputType:    "*map[string]string",
				expectedType: "*map[string]string",
				inputValue:   &map[string]string{"test": "test"},
				success:      true,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				ok, actualType := CheckType(testconfig.inputValue, testconfig.inputType)
				assert.Equalf(t, testconfig.expectedType, actualType, "Type not match ('%s' <>'%s)", actualType, testconfig.expectedType)
				assert.Equalf(t, testconfig.success, ok, "Result not match ('%v' <>'%v)", ok, testconfig.success)
			})
		}
		t.Run("Test Nil", func(t *testing.T) {
			var testMap *map[string]string
			ok, actualType := CheckType(testMap, "*map[string]string")
			assert.Equalf(t, "<nil>", actualType, "Type not match ('%s' <>'%s)", actualType, "<nil>")
			assert.Equalf(t, false, ok, "Result not match ('%v' <>'%v)", ok, false)
		})
	})
}

func TestIsNumeric(t *testing.T) {
	type testTableType struct {
		name     string
		input    string
		expected bool
	}
	for _, testconfig := range []testTableType{
		{
			name:     "isNumeric",
			input:    "123",
			expected: true,
		},
		{
			name:     "isNumericFloat",
			input:    "123.456",
			expected: true,
		},
		{
			name:     "isNumericNegative",
			input:    "-123",
			expected: true,
		},
		{
			name:     "isNumericNegativeFloat",
			input:    "+123.456",
			expected: true,
		},
		{
			name:     "isNumericEmpty",
			input:    "",
			expected: false,
		},
		{
			name:     "isNumericString",
			input:    "abc",
			expected: false,
		},
		{
			name:     "isNumericStringWithNumber",
			input:    "abc123",
			expected: false,
		},
		{
			name:     "isNumericStringWithNumberNegative",
			input:    "abc-123",
			expected: false,
		},
		{
			name:     "isNumericStringWithSpaces",
			input:    " 123 ",
			expected: false,
		},
	} {
		t.Run(testconfig.name, func(t *testing.T) {
			actual := IsNumeric(testconfig.input)
			assert.Equal(t, testconfig.expected, actual, "unexpected answer")
		})
	}
}
