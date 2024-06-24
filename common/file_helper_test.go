package common

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/test"
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

func TestFileHelper(t *testing.T) {
	test.InitTestDirs()
	t.Run("TestReadFileToString", func(t *testing.T) {
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
			var info os.FileInfo
			var content string
			err = ChdirToFile(filename)
			if err != nil {
				t.Fatalf("Cannot chdir to %s", filename)
			}
			wd, _ := os.Getwd()
			// use basename from filename to read as I am in this directory
			f := filepath.Base(filename)
			info, err = os.Stat(f)
			if err != nil {
				t.Fatalf("File %s/%s not found: %s", wd, f, err)
			}
			content, err = ReadFileToString(f)
			expected := info.Size()
			// need to convert to int64 type to be equal
			actual := int64(len(content))
			assert.NoErrorf(t, err, "Error: %s", err)
			assert.Equal(t, expected, actual, "Size mismatch, exp:%d, act:%d", expected, actual)
		})
	})

	t.Run("TestReadFileByLine", func(t *testing.T) {
		var content []string
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
			err = ChdirToFile(filename)
			if err != nil {
				t.Fatalf("Cannot chdir to %s", filename)
			}
			// use basename from filename to read as I am in this directory
			f := filepath.Base(filename)
			_, err = os.Stat(f)
			if err != nil {
				t.Fatalf("File %s/%s not found: %s", wd, f, err)
			}
			content, err = ReadFileByLine(f)
			expected := len(lines)
			actual := len(content)
			assert.NoErrorf(t, err, "Error: %s", err)
			assert.Equal(t, expected, actual, "line count mismatch, expected:%d, actual:%d", expected, actual)
		})
	})

	t.Run("TestChdirToFile", func(t *testing.T) {
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
			err = ChdirToFile("../xxx/yyy")
			assert.Errorf(t, err, "Chdir should fail")
		})
	})

	t.Run("TestFileExist", func(t *testing.T) {
		assert.True(t, FileExists(path.Join(test.TestDir, "testinit.go")))
		assert.False(t, FileExists("dummy.xxx"))
	})

	t.Run("TestCanRead", func(t *testing.T) {
		assert.True(t, CanRead(path.Join(test.TestDir, "testinit.go")))
		assert.False(t, CanRead("dummy.xxx"))
	})

	t.Run("TestIsDir", func(t *testing.T) {
		assert.True(t, IsDir(test.TestDir))
		assert.False(t, IsDir("dummy.xxx"))
		assert.False(t, IsDir(path.Join(test.TestDir, "testinit.go")))
	})

	t.Run("TestIsFile", func(t *testing.T) {
		assert.False(t, IsFile(test.TestDir))
		assert.False(t, IsFile("dummy.xxx"))
		assert.True(t, IsFile(path.Join(test.TestDir, "testinit.go")))
	})
}
