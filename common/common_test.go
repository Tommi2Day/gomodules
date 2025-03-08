package common

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	const fallback = "NotFound"
	t.Run("Test String Env", func(t *testing.T) {
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

		// Test with non-existing command
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

func TestRandString(t *testing.T) {
	t.Run("Test length of generated string", func(t *testing.T) {
		lengths := []int{0, 1, 5, 10, 100}
		for _, length := range lengths {
			result := RandString(length)
			assert.Equal(t, length, len(result), "Generated string length doesn't match expected length")
		}
	})

	t.Run("Test uniqueness of generated strings", func(t *testing.T) {
		length := 10
		iterations := 1000
		generatedStrings := make(map[string]bool)
		for i := 0; i < iterations; i++ {
			result := RandString(length)
			assert.False(t, generatedStrings[result], "Generated string is not unique")
			generatedStrings[result] = true
		}
	})

	t.Run("Test character set of generated string", func(t *testing.T) {
		length := 1000
		result := RandString(length)
		validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890"
		for _, char := range result {
			assert.Contains(t, validChars, string(char), "Generated string contains invalid character")
		}
	})

	t.Run("Test concurrent generation", func(t *testing.T) {
		length := 10
		iterations := 100
		results := make(chan string, iterations)

		var wg sync.WaitGroup
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results <- RandString(length)
			}()
		}

		wg.Wait()
		close(results)

		uniqueResults := make(map[string]bool)
		for result := range results {
			assert.False(t, uniqueResults[result], "Concurrent generation produced duplicate string")
			uniqueResults[result] = true
		}
	})
}

func TestStructToMap(t *testing.T) {
	t.Run("Test simple struct", func(t *testing.T) {
		type SimpleStruct struct {
			Name string
			Age  int
		}
		input := SimpleStruct{Name: "John", Age: 30}
		expected := map[string]interface{}{"Name": "John", "Age": float64(30)}

		result, err := StructToMap(input)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Test nested struct", func(t *testing.T) {
		type NestedStruct struct {
			Name    string
			Address struct {
				Street string
				City   string
			}
		}
		input := NestedStruct{
			Name: "Alice",
			Address: struct {
				Street string
				City   string
			}{
				Street: "123 Main St",
				City:   "Anytown",
			},
		}
		expected := map[string]interface{}{
			"Name": "Alice",
			"Address": map[string]interface{}{
				"Street": "123 Main St",
				"City":   "Anytown",
			},
		}

		result, err := StructToMap(input)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Test struct with slice", func(t *testing.T) {
		type SliceStruct struct {
			Name    string
			Numbers []int
		}
		input := SliceStruct{Name: "Bob", Numbers: []int{1, 2, 3}}
		expected := map[string]interface{}{
			"Name":    "Bob",
			"Numbers": []interface{}{float64(1), float64(2), float64(3)},
		}

		result, err := StructToMap(input)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Test struct with pointer", func(t *testing.T) {
		type PointerStruct struct {
			Name  string
			Value *int
		}
		value := 42
		input := PointerStruct{Name: "Charlie", Value: &value}
		expected := map[string]interface{}{"Name": "Charlie", "Value": float64(42)}

		result, err := StructToMap(input)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Test empty struct", func(t *testing.T) {
		type EmptyStruct struct{}
		input := EmptyStruct{}
		expected := map[string]interface{}{}

		result, err := StructToMap(input)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Test struct with unexported fields", func(t *testing.T) {
		type UnexportedStruct struct {
			Name string
			age  int
		}
		input := UnexportedStruct{Name: "Dave", age: 25}
		expected := map[string]interface{}{"Name": "Dave"}

		result, err := StructToMap(input)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Test non-struct input", func(t *testing.T) {
		input := "not a struct"
		_, err := StructToMap(input)
		assert.Error(t, err)
	})
}
func TestStructToString(t *testing.T) {
	t.Run("Test struct with multiple fields", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			Age   int
			Email string
		}
		testObj := TestStruct{
			Name:  "John Doe",
			Age:   30,
			Email: "john@example.com",
		}
		result := StructToString(testObj, "")
		expected := "Age: 30\nEmail: john@example.com\nName: John Doe\n"
		assert.Equal(t, expected, result, "Unexpected string representation")
	})

	t.Run("Test empty struct", func(t *testing.T) {
		type EmptyStruct struct{}
		testObj := EmptyStruct{}
		result := StructToString(testObj, "")
		assert.Empty(t, result, "Expected empty string for empty struct")
	})

	t.Run("Test struct with unexported fields", func(t *testing.T) {
		type TestStruct struct {
			Name string
			age  int
		}
		testObj := TestStruct{
			Name: "Alice",
			age:  25,
		}
		result := StructToString(testObj, "")
		expected := "Name: Alice\n"
		assert.Equal(t, expected, result, "Unexpected string representation")
	})

	t.Run("Test struct with pointer fields", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			Score *int
		}
		score := 100
		testObj := TestStruct{
			Name:  "Bob",
			Score: &score,
		}
		result := StructToString(testObj, "")
		expected := "Name: Bob\nScore: 100\n"
		assert.Equal(t, expected, result, "Unexpected string representation")
	})

	t.Run("Test non-struct input", func(t *testing.T) {
		testObj := "Not a struct"
		result := StructToString(testObj, "")
		assert.Empty(t, result, "Expected empty string for non-struct input")
	})

	t.Run("Test struct with nested struct", func(t *testing.T) {
		type Address struct {
			Street string
			City   string
		}
		type Person struct {
			Name    string
			Address Address
		}
		testObj := Person{
			Name: "Charlie",
			Address: Address{
				Street: "123 Main St",
				City:   "Anytown",
			},
		}
		result := StructToString(testObj, "")
		expected := "Address: \n  City: Anytown\n  Street: 123 Main St\n\nName: Charlie\n"
		assert.Equal(t, expected, result, "Unexpected string representation")
	})
}

// Merge two non-empty maps with non-overlapping keys
func TestMergeMaps(t *testing.T) {
	t.Run("TestMergeMapsWithNonOverlappingKeys", func(t *testing.T) {
		m1 := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}

		m2 := map[string]interface{}{
			"key3": true,
			"key4": []string{"a", "b"},
		}

		expected := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
			"key4": []string{"a", "b"},
		}

		result, err := MergeMaps(m1, m2)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		/*
			if !reflect.DeepEqual(result, expected) {
				t.Errorf("Expected %v but got %v", expected, result)
			}
		*/
	})

	// Merge nil map with non-nil map
	t.Run("TestMergeMapsWithNilMap", func(t *testing.T) {
		var m1 map[string]interface{}

		m2 := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}

		expected := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}

		result, err := MergeMaps(m1, m2)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	// Merge two non-empty maps with overlapping keys
	t.Run("TestMergeMapsWithOverlappingKeys", func(t *testing.T) {
		m1 := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		m2 := map[string]interface{}{
			"key2": "newValue2",
			"key3": "value3",
		}
		expected := map[string]interface{}{
			"key1": "value1",
			"key2": "newValue2",
			"key3": "value3",
		}
		result, err := MergeMaps(m1, m2)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	// Merge empty map with non-empty map
	t.Run("TestMergeEmptyMapWithNonEmptyMap", func(t *testing.T) {
		m1 := map[string]interface{}{}
		m2 := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		expected := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		result, err := MergeMaps(m1, m2)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	// Merge maps with nested interface{} values
	t.Run("TestMergeMapsWithNestedValues", func(t *testing.T) {
		m1 := map[string]interface{}{
			"key1": map[string]interface{}{
				"nestedKey1": "nestedValue1",
			},
		}
		m2 := map[string]interface{}{
			"key2": map[string]interface{}{
				"nestedKey2": "nestedValue2",
			},
		}
		expected := map[string]interface{}{
			"key1": map[string]interface{}{
				"nestedKey1": "nestedValue1",
			},
			"key2": map[string]interface{}{
				"nestedKey2": "nestedValue2",
			},
		}
		result, err := MergeMaps(m1, m2)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
	t.Run("TestMergeMapWithCustomType", func(t *testing.T) {
		type mytype map[string]string
		m1 := mytype{
			"key1": "value1",
			"key2": "value1",
		}
		m2 := mytype{
			"key1": "value1",
			"key2": "value2",
		}
		expected := mytype{
			"key1": "value1",
			"key2": "value2",
		}
		result, err := MergeMaps(m1, m2)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}
func TestMapToJson(t *testing.T) {
	// Convert a simple map to JSON string
	t.Run("TestMapToJsonWithSimpleMap", func(t *testing.T) {
		// Create a simple map
		testMap := map[string]interface{}{
			"name": "John",
			"age":  30,
			"city": "New York",
		}

		// Call the function
		result, err := StructToJSON(testMap)

		// Assert no error
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Expected JSON string
		expected := `{
 "age": 30,
 "city": "New York",
 "name": "John"
}`

		// Compare result with expected
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	// Handle nil input
	t.Run("TestMapToJsonWithNilInput", func(t *testing.T) {
		// Call the function with nil input
		result, err := StructToJSON(nil)

		// Assert no error
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Expected JSON string for nil
		expected := "null"

		// Compare result with expected
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	// Convert a struct to JSON string
	t.Run("TestConvertStructToJsonString", func(t *testing.T) {
		type Person struct {
			Name string
			Age  int
		}
		person := Person{Name: "Alice", Age: 30}
		expected := "{\n \"Name\": \"Alice\",\n \"Age\": 30\n}"

		jsonStr, err := StructToJSON(person)
		assert.NoError(t, err)
		assert.Equal(t, expected, jsonStr)
	})

	// Convert a nested map to JSON string
	t.Run("TestConvertNestedMapToJsonString", func(t *testing.T) {
		nestedMap := map[string]interface{}{
			"name": "Bob",
			"details": map[string]interface{}{
				"age":  25,
				"city": "New York",
			},
		}
		expected := "{\n \"details\": {\n  \"age\": 25,\n  \"city\": \"New York\"\n },\n \"name\": \"Bob\"\n}"

		jsonStr, err := StructToJSON(nestedMap)
		assert.NoError(t, err)
		assert.Equal(t, expected, jsonStr)
	})

	// Convert an array/slice to JSON string
	t.Run("TestConvertSliceToJsonString", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		expected := "[\n \"apple\",\n \"banana\",\n \"cherry\"\n]"
		jsonStr, err := StructToJSON(slice)
		assert.NoError(t, err)
		assert.Equal(t, expected, jsonStr)
	})
}
