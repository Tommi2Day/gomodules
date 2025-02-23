package pwlib

import (
	"crypto/rand"
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"
)

// MaxGenpassTrys defines the maximum number of attempts to generate a password meeting the specified constraints.
var MaxGenpassTrys = 200

// GenerateRandomString generate a randow string with the given length and out of allowed charset
func GenerateRandomString(length int, allowedChars string) string {
	letters := allowedChars
	if len(allowedChars) == 0 {
		letters = AllChars
		log.Debug("GenRandom: No allowed chars specified, using all chars")
	}
	ret := make([]byte, length)
	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		ret[i] = letters[num.Int64()]
	}
	log.Debugf("GenRandom candidate: %s", string(ret))
	return string(ret)
}

// GenPassword generates a password with the given complexity
func GenPassword(profile string) (string, error) {
	var err error
	pp := DefaultPasswordProfile
	if profile != "" {
		pp, err = GetPasswordProfileFromString(profile)
		if err != nil {
			return "", err
		}
	}
	pps := PasswordProfileSet{
		Profile: pp,
	}
	return GenPasswordProfile(pps)
}

// GenPasswordProfile generates a password based on the given PasswordProfileSet configuration.
// It ensures the generated password meets the specified constraints and attempts up to MaxGenpassTrys times.
// Returns the generated password and an error if unable to create a valid password.
func GenPasswordProfile(pps PasswordProfileSet) (string, error) {
	log.Debug("GenPassword entered")
	// allowed chars
	var ok = false
	var err error

	pp, cs := pps.Load()
	newPassword := ""
	// skip password check logging when used to generate
	SilentCheck = true
	// max 50 tries to generate a valid password
	for c := 0; c < MaxGenpassTrys; c++ {
		newPassword = GenerateRandomString(pp.Length, cs.AllChars)
		ok = DoPasswordCheck(newPassword, pp, cs)
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
