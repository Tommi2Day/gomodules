package pwlib

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// DoPasswordCheck Checks a password to given criteria
func DoPasswordCheck(password string, length int, upper int, lower int, numeric int, special int, firstCharCheck bool, allowedChars string) bool {
	// var ls = true
	var ucs = true
	var lcs = true
	var ncs = true
	var sps = true
	// var cs = true
	var fcs = true
	var err error

	// allowed chars
	possible := allowedChars
	if allowedChars == "" {
		possible = AllChars
	}

	// do checks
	ls, err := checkLength(password, length)
	debugLog("length", err)
	if upper > 0 {
		ucs, err = checkClass(password, upper, UpperChar)
		debugLog("uppercase", err)
	}
	if lower > 0 {
		lcs, err = checkClass(password, lower, LowerChar)
		debugLog("lowercase", err)
	}
	if numeric > 0 {
		ncs, err = checkClass(password, numeric, Digits)
		debugLog("numeric", err)
	}
	if special > 0 {
		sps, err = checkClass(password, special, SpecialChar)
		debugLog("special", err)
	}
	cs, err := checkChars(password, possible)
	debugLog("allowed chars", err)
	if firstCharCheck {
		fcs, err = checkFirstChar(password, UpperChar+LowerChar)
		debugLog("first character", err)
	}

	// final state
	return ls && ucs && lcs && ncs && sps && cs && fcs
}

func debugLog(name string, err error) {
	if err != nil {
		log.Debugf("%s check failed,", err.Error())
	} else {
		log.Debugf("%s check passed", name)
	}
}

func checkClass(
	password string,
	should int,
	chars string,
) (bool, error) {
	if len(password) == 0 {
		return false, fmt.Errorf("password empty")
	}
	cnt := 0
	for _, char := range strings.Split(chars, "") {
		cnt += strings.Count(password, char)
	}
	if cnt < should {
		return false, fmt.Errorf("at least %d chars from %s", should, chars)
	}
	return true, nil
}

func checkChars(
	password string,
	chars string,
) (bool, error) {
	if len(password) == 0 {
		return false, fmt.Errorf("%s check failed, password empty", "character")
	}
	data := []rune(password)
	for i := 0; i < len(data); i++ {
		r := data[i]
		idx := strings.IndexRune(chars, r)
		if idx == -1 {
			return false, fmt.Errorf("%s check failed, only %s allowed", "character", chars)
		}
	}
	return true, nil
}

func checkLength(password string, minlen int) (bool, error) {
	length := len(password)
	if length < minlen {
		return false, fmt.Errorf("length check failed, at least  %d chars expected, have %d", minlen, length)
	}
	return true, nil
}

func checkFirstChar(
	password string,
	allowed string,
) (bool, error) {
	if len(password) == 0 {
		return false, fmt.Errorf("%s check failed, password empty", "first letter")
	}
	firstLetter := []rune(password)[0]
	idx := strings.IndexRune(allowed, firstLetter)
	if idx == -1 {
		return false, fmt.Errorf("%s check failed, only %s allowed", "first letter", allowed)
	}
	return true, nil
}
