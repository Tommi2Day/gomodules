package pwlib

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// DoPasswordCheck Checks a password to given criteria
//
//gocyclo:ignore
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
	if err != nil {
		log.Debug(err.Error())
	}
	if upper > 0 {
		r, err := checkClass(password, upper, UpperChar, "uppercase")
		ucs = r
		if err != nil {
			log.Debug(err.Error())
		} else {
			log.Debug("UpperCase check passed")
		}
	}
	if lower > 0 {
		r, err := checkClass(password, lower, LowerChar, "lowercase")
		lcs = r
		if err != nil {
			log.Debug(err.Error())
		} else {
			log.Debug("LowerCase check passed")
		}
	}
	if numeric > 0 {
		r, err := checkClass(password, numeric, Digits, "numeric")
		ncs = r
		if err != nil {
			log.Debug(err.Error())
		} else {
			log.Debug("Digits check passed")
		}
	}
	if special > 0 {
		r, err := checkClass(password, special, SpecialChar, "special")
		sps = r
		if err != nil {
			log.Debug(err.Error())
		} else {
			log.Debug("SpecialChar check passed")
		}
	}
	cs, err := checkChars(password, possible)
	if err != nil {
		log.Debug(err.Error())
	} else {
		log.Debug("allowed Char check passed")
	}
	if firstCharCheck {
		r, err := checkFirstChar(password, UpperChar+LowerChar)
		if err != nil {
			log.Debug(err.Error())
		} else {
			log.Debug("FirstChar check passed")
		}
		fcs = r
	}

	// final state
	return ls && ucs && lcs && ncs && sps && cs && fcs
}

func checkClass(
	password string,
	should int,
	chars string,
	name string,
) (bool, error) {
	if len(password) == 0 {
		return false, fmt.Errorf("%s check failed, password empty", name)
	}
	cnt := 0
	for _, char := range strings.Split(chars, "") {
		cnt += strings.Count(password, char)
	}
	if cnt < should {
		return false, fmt.Errorf("%s check failed, at least %d chars from %s", name, should, chars)
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
