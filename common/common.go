// Package common collection of small utils
package common

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"time"
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

// GetStringEnv read an OS Env variable
func GetStringEnv(key string, fallback string) string {
	return GetEnv(key, fallback)
}

// GetBoolEnv read an OS Env variable and convert to bool
func GetBoolEnv(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, err := GetBoolVal(value); err == nil {
			return b
		}
	}
	return fallback
}

// GetIntEnv read an OS Env variable and convert to int
func GetIntEnv(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := GetIntVal(value); err == nil {
			return i
		}
	}
	return fallback
}

// GetFloatEnv read an OS Env variable and convert to float64
func GetFloatEnv(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := GetFloatVal(value); err == nil {
			return i
		}
	}
	return fallback
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
	//nolint gosec
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

// InArray will search element inside array with any type.
// Will return boolean and index for matched element.
// True and index more than 0 if element is exist.
// needle is element to search, haystack is slice of value to be search.
// Taken from https://github.com/SimonWaldherr/golang-examples/blob/master/advanced/in_array.go
func InArray(needle interface{}, haystack interface{}) (exists bool, index int) {
	exists = false
	index = -1
	if reflect.TypeOf(haystack).Kind() == reflect.Slice {
		s := reflect.ValueOf(haystack)
		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(needle, s.Index(i).Interface()) {
				index = i
				exists = true
				return
			}
		}
	}
	return
}

// FormatUnixtsString converts a unix timestamp string to a human readable
func FormatUnixtsString(ts string, layout string) string {
	if !IsNumeric(ts) {
		return ts
	}
	timestamp, err := GetInt64Val(ts)
	if err != nil {
		return ts
	}
	return time.Unix(timestamp, 0).Format(layout)
}
