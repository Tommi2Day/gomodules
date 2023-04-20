package pwlib

import (
	"crypto/rand"
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"
)

// generateRandomString generate a randow string with the given length and out of allowed charset
func generateRandomString(length int, allowedChars string) string {
	letters := allowedChars
	ret := make([]byte, length)
	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		ret[i] = letters[num.Int64()]
	}
	log.Debugf("Password candidate: %s", string(ret))
	return string(ret)
}

// GenPassword generates a password with the given complexity
func GenPassword(length int, upper int, lower int, numeric int, special int, firstCharCheck bool) (string, error) {
	log.Debug("GenPassword entered")
	// allowed chars
	var ok = false
	var err error

	var allowedChars = ""
	// define allowed charset based on parameters
	if upper > 0 {
		allowedChars += charset.UpperChar
	}
	if lower > 0 {
		allowedChars += charset.LowerChar
	}
	if numeric > 0 {
		allowedChars += charset.Digits
	}
	if special > 0 {
		allowedChars += charset.SpecialChar
	}
	charset.AllChars = allowedChars

	newPassword := ""
	// skip password check logging when used to generate
	SilentCheck = true
	// max 50 tries to generate a valid password
	for c := 0; c < 50; c++ {
		newPassword = generateRandomString(length, allowedChars)
		ok = DoPasswordCheck(newPassword, length, upper, lower, numeric, special, firstCharCheck, allowedChars)
		if ok {
			break
		}
		newPassword = ""
		log.Debugf("generate retry %d", c)
	}
	SilentCheck = false

	if !ok {
		err = fmt.Errorf("unable to create required Password")
	} else {
		log.Debug("Generation succeeded")
	}
	return newPassword, err
}
