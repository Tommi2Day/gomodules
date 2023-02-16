// Package common collection of small utils
package common

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	log "github.com/sirupsen/logrus"
)

func init() {
	// use text formatter
	// log.SetFormatter(&log.TextFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.WarnLevel)
}

// GetEnv read an OS Env variable
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// ReadFileToString read a file and return a string
func ReadFileToString(filename string) (string, error) {
	filename = filepath.Clean(filename)
	if _, err := os.Stat(filename); err != nil {
		return "", fmt.Errorf("file %s  not found", filename)
	}
	//nolint gosec
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Debugf("Error closing " + filename)
		}
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

// ReadFileByLine read a file and return array of lines
func ReadFileByLine(filename string) ([]string, error) {
	var lines []string
	filename = filepath.Clean(filename)
	if _, err := os.Stat(filename); err != nil {
		return lines, fmt.Errorf("file %s  not found", filename)
	}
	//nolint gosec
	f, err := os.Open(filename)
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Debugf("Error closing " + filename)
		}
	}(f)

	if err != nil {
		return lines, err
	}
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
	} else {
		log.Warnf(" >Read Failed!: %v\n", err)
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

// CheckSkip checks if a line can be skipped
func CheckSkip(line string) (skip bool) {
	skip = true
	found := false
	reEmpty := regexp.MustCompile(`\S`)
	reComment := regexp.MustCompile(`^#`)
	found = reEmpty.MatchString(line)
	if !found {
		return
	}
	found = reComment.MatchString(line)
	if found {
		return
	}
	skip = false
	return
}

// RemoveSpace deletes all spaces and newlines from string
func RemoveSpace(s string) string {
	rr := make([]rune, 0, len(s))
	for _, r := range s {
		if !unicode.IsSpace(r) {
			rr = append(rr, r)
		}
	}
	return string(rr)
}

// ExecuteOsCommand runs an OS command and returns output
func ExecuteOsCommand(cmdArgs []string, stdIn io.Reader) (stdOut string, stdErr string, err error) {
	var cmdOut bytes.Buffer
	var cmdErr bytes.Buffer
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmdOut.Reset()
	cmdErr.Reset()
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	cmd.Stdin = stdIn
	err = cmd.Run()
	stdOut = cmdOut.String()
	stdErr = cmdErr.String()
	return
}
