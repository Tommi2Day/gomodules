package common

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ReadFileToString read a file and return a string
func ReadFileToString(filename string) (string, error) {
	filename = filepath.Clean(filename)
	if !FileExists(filename) {
		return "", fmt.Errorf("file %s  not found", filename)
	}
	//nolint gosec
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	b := new(strings.Builder)
	fi, _ := f.Stat()
	b.Grow(int(fi.Size()))
	_, err = io.Copy(b, f)
	if err != nil {
		return "", err
	}
	return b.String(), err
}

// ReadStdinToString reads stdin and returns a string
func ReadStdinToString() (string, error) {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(stdin), nil
}

// WriteStringToFile saves a string to filename and assign rights 0600
func WriteStringToFile(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0600)
}

// ReadFileByLine read a file and return array of lines
func ReadFileByLine(filename string) ([]string, error) {
	var lines []string
	filename = filepath.Clean(filename)
	if !FileExists(filename) {
		return lines, fmt.Errorf("file %s  not found", filename)
	}
	//nolint gosec
	f, err := os.Open(filename)
	if err != nil {
		return lines, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	var line string
	reader := bufio.NewReader(f)
	for {
		line, err = reader.ReadString('\n')
		lines = append(lines, line)
		if err != nil {
			break
		}
	}

	if err == io.EOF {
		err = nil
	}
	return lines, err
}

// ReadStdinByLine reads stdin and returns array of lines
func ReadStdinByLine() ([]string, error) {
	var lines []string
	var err error
	reader := bufio.NewReader(os.Stdin)
	for {
		line := ""
		line, err = reader.ReadString('\n')
		if err != nil {
			if line != "" {
				lines = append(lines, line)
			}
			break
		}
		lines = append(lines, line)
	}
	if err == io.EOF {
		err = nil
	}
	return lines, err
}

// ChdirToFile change working directory to the filename
func ChdirToFile(file string) error {
	a, _ := filepath.Abs(file)
	d := filepath.Dir(a)
	err := os.Chdir(d)
	log.Debugf("chdir to %s\n", d)
	return err
}

// FileExists checks if a file or directory exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("file %s does not exist\n", filename)
			return false
		}
		log.Debugf("file stat problem for %s:%s\n", filename, err)
		return false
	}
	log.Debugf("file %s exists\n", filename)
	return true
}

// CanRead checks if a file can be read(opened
func CanRead(filename string) bool {
	//nolint gosec
	_, err := os.Open(filename)
	return err == nil
}

// IsDir checks if a filename is a directory
func IsDir(name string) bool {
	fi, err := os.Stat(name)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// IsFile checks if a filename is a file
func IsFile(name string) bool {
	fi, err := os.Stat(name)
	if err != nil {
		return false
	}
	return !fi.IsDir()
}
