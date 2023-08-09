package common

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tommi2day/gomodules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const data = `
# Testfile
!default:defuser2:failure
!default:testuser:default
test:testuser:testpass
testdp:testuser:xxx:yyy
!default:defuser2:default
!default:testuser:failure
!default:defuser:default
`

func TestReadFileToString(t *testing.T) {
	test.Testinit(t)
	// prepare
	filename := "testdata/stringtest.test"
	err := os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	_ = os.Remove(filename)
	//nolint gosec
	err = os.WriteFile(filename, []byte(data), 0644)
	require.NoErrorf(t, err, "Create testdata failed")

	// run
	t.Run("Read File to String", func(t *testing.T) {
		err := ChdirToFile(filename)
		if err != nil {
			t.Fatalf("Cannot chdir to %s", filename)
		}
		wd, _ := os.Getwd()
		// use basename from filename to read as I am in this directory
		f := filepath.Base(filename)
		info, err := os.Stat(f)
		if err != nil {
			t.Fatalf("File %s/%s not found: %s", wd, f, err)
		}
		content, err := ReadFileToString(f)
		expected := info.Size()
		// need to convert to int64 type to be equal
		actual := int64(len(content))
		assert.NoErrorf(t, err, "Error: %s", err)
		assert.Equal(t, expected, actual, "Size mismatch, exp:%d, act:%d", expected, actual)
	})
}

func TestReadFileByLine(t *testing.T) {
	test.Testinit(t)
	// prepare
	filename := "testdata/linetest.test"
	err := os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	_ = os.Remove(filename)
	//nolint gosec
	err = os.WriteFile(filename, []byte(data), 0644)
	require.NoErrorf(t, err, "Create testdata failed")
	lines := strings.Split(data, "\n")

	// run
	t.Run("Read File By Lines", func(t *testing.T) {
		wd, _ := os.Getwd()
		t.Logf("work in %s", wd)
		err := ChdirToFile(filename)
		if err != nil {
			t.Fatalf("Cannot chdir to %s", filename)
		}
		// use basename from filename to read as I am in this directory
		f := filepath.Base(filename)
		_, err = os.Stat(f)
		if err != nil {
			t.Fatalf("File %s/%s not found: %s", wd, f, err)
		}
		content, err := ReadFileByLine(f)
		expected := len(lines)
		actual := len(content)
		assert.NoErrorf(t, err, "Error: %s", err)
		assert.Equal(t, expected, actual, "line count mismatch, expected:%d, actual:%d", expected, actual)
	})
}

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

func TestChdirToFile(t *testing.T) {
	err := os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	filename := "testdata/chdir.test"
	wd, _ := os.Getwd()
	t.Logf("work in %s", wd)
	full := filepath.Clean(wd + "/" + filename)
	expected := filepath.Dir(full)

	t.Run("Test chDir", func(t *testing.T) {
		err := ChdirToFile(filename)
		actual, _ := os.Getwd()
		assert.NoErrorf(t, err, "Chdir failed")
		assert.NotEmpty(t, actual, "WD Empty")
		assert.Equalf(t, expected, actual, "value not expected E:%s, A:%s", expected, actual)
	})
	t.Run("Test nonexisting Dir", func(t *testing.T) {
		err := ChdirToFile("../xxx/yyy")
		assert.Errorf(t, err, "Chdir should fail")
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
	case "windows":
		cmdArg = []string{"cmd.exe", "/c", "dir"}
	default:
		cmdArg = []string{"/bin/ls"}
	}

	stdout, stderr, err := ExecuteOsCommand(cmdArg, nil)
	assert.NoErrorf(t, err, "Command got Error: %v", err)
	assert.Emptyf(t, stderr, "StdErr not empty")
	assert.NotEmpty(t, stdout, "Output is empty")
}

func TestConversion(t *testing.T) {
	t.Run("Check Host parsing", func(t *testing.T) {
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
func TestGetHostPort(t *testing.T) {
	t.Run("Check Host parsing", func(t *testing.T) {
		type testTableType struct {
			name    string
			input   string
			success bool
			host    string
			port    int
		}
		for _, testconfig := range []testTableType{
			{
				name:    "only host",
				input:   "localhost",
				success: false,
				host:    "localhost",
				port:    0,
			},
			{
				name:    "host and port",
				input:   "localhost:1234",
				success: true,
				host:    "localhost",
				port:    1234,
			},
			{
				name:    "with tcp url",
				input:   "tcp://docker:2375",
				success: true,
				host:    "docker",
				port:    2375,
			},
			{
				name:    "with http url",
				input:   "http://localhost:8080/app/index.html",
				success: true,
				host:    "localhost",
				port:    8080,
			},
			{
				name:    "with http url witout port",
				input:   "http://localhost/app/index.html",
				success: true,
				host:    "localhost",
				port:    80,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				host, port, err := GetHostPort(testconfig.input)
				if testconfig.success {
					assert.NoErrorf(t, err, "unexpected error %s", err)
					assert.Equalf(t, testconfig.host, host, "entry returned wrong host ('%s' <>'%s)", host, testconfig.host)
					assert.Equalf(t, testconfig.port, port, "entry returned wrong port ('%d' <>'%d)", port, testconfig.port)
				} else {
					assert.Error(t, err, "Expected error not set")
				}
			})
		}
	})
}
func TestSetHostPort(t *testing.T) {
	t.Run("Test SetHostPort ipv4", func(t *testing.T) {
		actual := SetHostPort("localhost", 1234)
		assert.Equalf(t, "localhost:1234", actual, "actual not expected %s", actual)
	})
	t.Run("Test SetHostPort tcpv6", func(t *testing.T) {
		actual := SetHostPort("fe80::3436:bd7c:3037:df6f", 1234)
		assert.Equalf(t, "[fe80::3436:bd7c:3037:df6f]:1234", actual, "actual not expected: %s", actual)
	})
	t.Run("Test SetHostPort noport", func(t *testing.T) {
		actual := SetHostPort("localhost", 0)
		assert.Equalf(t, "localhost", actual, "actual not expected: %s", actual)
	})
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
