package common

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

// GetBoolVal convert String to bool
func GetBoolVal(value string) (bool, error) {
	return strconv.ParseBool(value)
}

// GetInt64Val convert String to int64
func GetInt64Val(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

// GetIntVal convert String to int
func GetIntVal(value string) (int, error) {
	v, err := strconv.ParseInt(value, 10, 32)
	return int(v), err
}

// GetFloatVal convert String to float64
func GetFloatVal(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// IsNil checks if an interface is nil
func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	//nolint exhaustive
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

// IsNumeric checks if a trimmed string is numeric using regexp
func IsNumeric(s string) bool {
	re := regexp.MustCompile(`^[+-\\d.]+$`)
	r := re.MatchString(s)
	return r
}

// CheckType checks if an interface is of a certain type and returns if it matches expected or is Nil
func CheckType(t any, expected string) (ok bool, haveType string) {
	ok = false
	haveType = fmt.Sprintf("%T", t)
	if haveType != expected {
		return
	}
	if IsNil(t) {
		haveType = "<nil>"
		return
	}
	ok = true
	return
}
