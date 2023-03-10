// Package test defines path settings while testing
package test

// https://intellij-support.jetbrains.com/hc/en-us/community/posts/360009685279-Go-test-working-directory-keeps-changing-to-dir-of-the-test-file-instead-of-value-in-template
import (
	"os"
	"path"
	"runtime"
)

// TestDir working dir for tests
var TestDir string

// TestData directory for working files
var TestData string

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Dir(filename)
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
	TestDir = dir
	TestData = path.Join(TestDir, "testdata")
	// create data directory and ignore errors
	err = os.Mkdir(TestData, 0750)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
	println("Work in " + dir)
}
