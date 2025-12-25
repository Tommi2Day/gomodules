package common

import (
	"encoding/xml"
	"os"
	"path"
	"path/filepath"
	"runtime"
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

		err = WriteStringToFile(filename, data)
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

		err = WriteStringToFile(filename, data)
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

func TestWriteStringToFile(t *testing.T) {
	test.InitTestDirs()
	t.Run("Write content to new file", func(t *testing.T) {
		filename := test.TestData + "/writetest_new.txt"
		content := "This is a test content"

		// write content
		err := WriteStringToFile(filename, content)
		assert.NoError(t, err)

		// content test
		readContent, err := ReadFileToString(filename)
		assert.NoError(t, err)
		assert.Equal(t, content, readContent)

		// mode test (not in windows
		if runtime.GOOS != osWin {
			fileStat, err := os.Stat(filename)
			expectedMode := os.FileMode.Perm(0600)
			assert.NoError(t, err)
			assert.Equalf(t, expectedMode, fileStat.Mode(), "File permissions '%s' not as expected '%s'", expectedMode, fileStat.Mode())
		}
	})

	t.Run("Overwrite existing file", func(t *testing.T) {
		filename := test.TestData + "/writetest_existing.txt"
		initialContent := "Initial content"
		newContent := "New content"

		err := WriteStringToFile(filename, initialContent)
		assert.NoError(t, err)

		err = WriteStringToFile(filename, newContent)
		assert.NoError(t, err)

		readContent, err := ReadFileToString(filename)
		assert.NoError(t, err)
		assert.Equal(t, newContent, readContent)
	})

	t.Run("Write empty string", func(t *testing.T) {
		filename := test.TestData + "/writetest_empty.txt"
		content := ""

		err := WriteStringToFile(filename, content)
		assert.NoError(t, err)

		readContent, err := ReadFileToString(filename)
		assert.NoError(t, err)
		assert.Empty(t, readContent)
	})

	t.Run("Write to file in non-existent directory", func(t *testing.T) {
		filename := test.TestData + "/nonexistent/writetest.txt"
		content := "Test content"

		err := WriteStringToFile(filename, content)
		assert.Error(t, err)
	})
}

func TestReadStdinByLine(t *testing.T) {
	t.Run("Read multiple lines from stdin", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		input := "line1\nline2\nline3\n"
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdin = r
		go func() {
			_, _ = w.Write([]byte(input))
			_ = w.Close()
		}()

		lines, err := ReadStdinByLine()
		assert.NoError(t, err)
		assert.Equal(t, 3, len(lines))
		assert.Equal(t, "line1\n", lines[0])
		assert.Equal(t, "line2\n", lines[1])
		assert.Equal(t, "line3\n", lines[2])
	})
	t.Run("Read empty input", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		r, w, err := os.Pipe()
		require.NoError(t, err)

		os.Stdin = r
		go func() {
			_ = w.Close()
		}()

		lines, err := ReadStdinByLine()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(lines))
	})

	t.Run("Read input without final newline", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		input := "line1\nline2\nline3"
		r, w, err := os.Pipe()
		require.NoError(t, err)

		os.Stdin = r
		go func() {
			_, _ = w.Write([]byte(input))
			_ = w.Close()
		}()

		lines, err := ReadStdinByLine()
		assert.NoError(t, err)
		assert.Equal(t, 3, len(lines))
		assert.Equal(t, "line1\n", lines[0])
		assert.Equal(t, "line2\n", lines[1])
		assert.Equal(t, "line3", lines[2])
	})
}
func TestReadStdinToString(t *testing.T) {
	t.Run("Read valid input from stdin", func(t *testing.T) {
		// Save original stdin
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		expected := "test input\nwith multiple lines\n"
		r, w, err := os.Pipe()
		require.NoError(t, err)

		os.Stdin = r
		go func() {
			_, _ = w.Write([]byte(expected))
			_ = w.Close()
		}()

		result, err := ReadStdinToString()
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Read empty input from stdin", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		r, w, err := os.Pipe()
		require.NoError(t, err)

		os.Stdin = r
		go func() {
			_ = w.Close()
		}()

		result, err := ReadStdinToString()
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Handle large input from stdin", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		expected := strings.Repeat("a", 1024*1024) // 1MB of data
		r, w, err := os.Pipe()
		require.NoError(t, err)

		os.Stdin = r
		go func() {
			_, _ = w.Write([]byte(expected))
			_ = w.Close()
		}()

		result, err := ReadStdinToString()
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Handle closed stdin", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		r, w, err := os.Pipe()
		require.NoError(t, err)
		_ = w.Close()
		_ = r.Close()
		os.Stdin = r

		result, err := ReadStdinToString()
		assert.Error(t, err)
		assert.Empty(t, result)
	})
}
func TestReadFileToStruct(t *testing.T) {
	test.InitTestDirs()
	type TestStruct struct {
		XMLName xml.Name `xml:"test"`
		Name    string   `json:"name" yaml:"name" xml:"name"`
		Value   int      `json:"value" yaml:"value" xml:"value"`
		Enabled bool     `json:"enabled" yaml:"enabled" xml:"enabled,attr"`
	}
	tests := []struct {
		name     string
		content  string
		filename string
		wantErr  bool
		data     interface{}
	}{
		{
			name:     "Valid JSON file",
			filename: test.TestData + "/valid.json",
			content:  `{"name": "test", "value": 10, "enabled": true}`,
			data:     &TestStruct{},
			wantErr:  false,
		},
		{
			name:     "Valid YAML file",
			filename: test.TestData + "/valid.yaml",
			content:  "name: test\nvalue: 10\nenabled: true",
			data:     &TestStruct{},
			wantErr:  false,
		},
		{
			name:     "Valid XML file",
			filename: test.TestData + "/valid.xml",
			content:  `<test enabled="true">\n<name>test</name>\n<value>10</value>\n</test>`,
			data:     &TestStruct{},
			wantErr:  false,
		},

		{
			name:     "invalid json",
			filename: test.TestData + "/invalid.json",
			content:  `{"name": "test", "value": invalid, "enabled": true}`,
			data:     &TestStruct{},
			wantErr:  true,
		},
		{
			name:     "invalid yaml",
			filename: test.TestData + "/invalid.yaml",
			content:  "name: test\nvalue: 'invalid': true",
			data:     &TestStruct{},
			wantErr:  true,
		},
		{
			name:     "invalid xml",
			filename: test.TestData + "/invalid.xml",
			content:  `<test><name>test</name><value>invalid</invalid></test>`,
			data:     &TestStruct{},
			wantErr:  true,
		},
		{
			name:     "nil pointer",
			filename: test.TestData + "/invalid.xml",
			content:  `<test><name>test</name><value>10</invalid></test>`,
			data:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove(tt.filename)
			err := WriteStringToFile(tt.filename, tt.content)
			if err != nil {
				t.Errorf("error writing file: %v", err)
				return
			}
			val := tt.data
			err = ReadFileToStruct(tt.filename, val)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFileToStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.NotNil(t, val)
				d, ok := val.(*TestStruct)
				assert.True(t, ok)
				if ok {
					assert.Equal(t, "test", d.Name)
					assert.Equal(t, 10, d.Value)
					assert.Equal(t, true, d.Enabled)
				}
			}
		})
	}
}

func TestReadFileToStructComplex(t *testing.T) {
	test.InitTestDirs()
	type Address struct {
		Street     string `json:"street,omitempty" yaml:"street,omitempty" xml:"street,omitempty"`
		City       string `json:"city" yaml:"city" xml:"city"`
		PostalCode string `json:"postal_code,omitempty" yaml:"postal_code,omitempty" xml:"postal_code,omitempty"`
	}

	type Person struct {
		FirstName string  `json:"first_name,omitempty" yaml:"first_name,omitempty" xml:"first_name,omitempty"`
		LastName  string  `json:"last_name" yaml:"last_name" xml:"last_name"`
		Age       int     `json:"age,omitempty" yaml:"age,omitempty" xml:"age,omitempty"`
		Address   Address `json:"address" yaml:"address" xml:"address"`
	}
	expected := &Person{
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
		Address: Address{
			Street:     "123 Main St",
			City:       "Anytown",
			PostalCode: "12345",
		},
	}
	tests := []struct {
		name     string
		content  string
		filename string
		wantErr  bool
		data     interface{}
		expected *Person
	}{
		{
			name:     "Valid JSON file",
			filename: test.TestData + "/valid2.json",
			content: `{
			"first_name": "John",
			"last_name": "Doe",
			"age": 30,
			"address": {
				"street": "123 Main St",
				"city": "Anytown",
				"postal_code": "12345"
				}
		    }`,
			data:     &Person{},
			wantErr:  false,
			expected: expected,
		},
		{
			name:     "Valid YAML file",
			filename: test.TestData + "/valid2.yaml",
			content: `---
first_name: John
last_name: Doe
age: 30
address:
    street: "123 Main St"
    city: "Anytown"
    postal_code: "12345"
`,
			data:     &Person{},
			wantErr:  false,
			expected: expected,
		},
		{
			name:     "Valid XML file",
			filename: test.TestData + "/valid2.xml",
			content: `<?xml version="1.0" encoding="UTF-8"?>
		<person>
		<first_name>John</first_name>
		<last_name>Doe</last_name>
		<age>30</age>
		<address>
		<street>123 Main St</street>
		<city>Anytown</city>
		<postal_code>12345</postal_code>
		</address>
		</person>`,
			data:     &Person{},
			wantErr:  false,
			expected: expected,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove(tt.filename)
			err := WriteStringToFile(tt.filename, tt.content)
			if err != nil {
				t.Errorf("error writing file: %v", err)
				return
			}
			val := tt.data
			err = ReadFileToStruct(tt.filename, val)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFileToStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.NotNil(t, val)
				assert.Equal(t, tt.expected, val)
			}
		})
	}
}

func TestFindFile(t *testing.T) {
	t.Run("TestFindFileFileExists", func(t *testing.T) {
		// Create temp test directory
		tmpDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatal(err)
		}

		// Create test file
		testFile := filepath.Join(tmpDir, "test.txt")
		if err = os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		dirs := []string{tmpDir}
		result := FindFileInPath("test.txt", dirs)

		expected, _ := filepath.Abs(testFile)
		assert.NotEmpty(t, result)
		assert.Equalf(t, expected, result, "Expected %s, got %s", expected, result)
		_ = os.RemoveAll(tmpDir)
	})

	t.Run("TestFindFileFromRef", func(t *testing.T) {
		// Create temp test directory
		tmpDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatal(err)
		}

		// Create test file
		testFile := filepath.Join(tmpDir, "test.txt")
		if err = os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
		// Create reference file
		refFile := filepath.Join(tmpDir, "ref.txt")
		if err = os.WriteFile(refFile, []byte("ref"), 0600); err != nil {
			t.Fatal(err)
		}
		dirs := []string{refFile}
		result := FindFileInPath("test.txt", dirs)

		expected, _ := filepath.Abs(testFile)
		assert.NotEmpty(t, result)
		assert.Equalf(t, expected, result, "Expected %s, got %s", expected, result)

		_ = os.RemoveAll(tmpDir)
	})
	// Return empty string when file is not found in any directory
	t.Run("TestFindFileFileNotFound", func(t *testing.T) {
		// Create temp test directory
		tmpDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatal(err)
		}

		dirs := []string{tmpDir}
		result := FindFileInPath("nonexistent.txt", dirs)

		assert.Empty(t, result)
		_ = os.RemoveAll(tmpDir)
	})
}
