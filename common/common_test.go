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
