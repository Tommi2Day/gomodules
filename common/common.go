// Package common collection of small utils
package common

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	log "github.com/sirupsen/logrus"
)

const osWin = "windows"

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

// CommandExists checks if a command exists in PATH
func CommandExists(command string) bool {
	log.Debugf("checking if command %s exists", command)
	p, err := exec.LookPath(command)
	if err != nil {
		log.Debugf("%s not found in PATH: %v", command, err)
		return false
	}
	log.Debugf("command %s found as: %s", command, p)
	return true
}

// FindCommand searches for a command in the PATH environment variable
func FindCommand(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
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

// ReverseMap will reverse a map
func ReverseMap[M ~map[K]V, K comparable, V comparable](m M) map[V]K {
	reversedMap := make(map[V]K)
	for key, value := range m {
		reversedMap[value] = key
	}
	return reversedMap
}

// StructToMap converts a struct to a map
func StructToMap(obj interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// StructToString converts a struct to sorted key string
func StructToString(o interface{}, prefix string) (s string) {
	m, err := StructToMap(o)
	if err != nil {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := m[k]
		t := fmt.Sprintf("%v", reflect.TypeOf(v))
		if strings.HasPrefix(t, "map") {
			s += fmt.Sprintf("%s: \n%s\n", k, StructToString(v, "  "))
		} else {
			s += fmt.Sprintf("%s%s: %v\n", prefix, k, m[k])
		}
	}
	return s
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

// RandString generates a pseudo random string with letters and digits with desired length
func RandString(n int) string {
	const validChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(validChars))))
		ret[i] = validChars[num.Int64()]
	}
	return string(ret)
}

// MergeMapsAny merges two maps of any types, inserting all key-value pairs from the second map into the first map, and returns the updated map.
// Note: Both input maps must be of the same type.
func MergeMaps(m1, m2 interface{}) (interface{}, error) {
	if m1 == nil {
		return m2, nil
	}
	if m2 == nil {
		return m1, nil
	}

	map1Value := reflect.ValueOf(m1)
	map2Value := reflect.ValueOf(m2)

	// Ensure both inputs are maps
	if map1Value.Kind() != reflect.Map || map2Value.Kind() != reflect.Map {
		return nil, fmt.Errorf("both inputs must be maps")
	}

	// Ensure maps have the same key and value types
	if map1Value.Type() != map2Value.Type() {
		return nil, fmt.Errorf("maps must be of the same type")
	}

	mergedMap := reflect.MakeMap(map1Value.Type())

	// Copy elements from map1 to mergedMap
	for _, key := range map1Value.MapKeys() {
		mergedMap.SetMapIndex(key, map1Value.MapIndex(key))
	}

	// Copy elements from map2 to mergedMap
	for _, key := range map2Value.MapKeys() {
		mergedMap.SetMapIndex(key, map2Value.MapIndex(key))
	}

	return mergedMap.Interface(), nil
}

func StructToJSON(m interface{}) (string, error) {
	b, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		return "", err
	}
	c := string(b)
	return c, nil
}
