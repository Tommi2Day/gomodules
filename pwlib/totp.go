package pwlib

import (
	"fmt"

	"github.com/xlzd/gotp"
)

// GetOtp calculates a standard 6 digit TOTP from given secret
func GetOtp(secret string) (val string, e error) {
	e = nil
	// trap panics
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("panic:%v", err)
		}
	}()
	otp := gotp.NewDefaultTOTP(secret)
	if otp != nil {
		val = otp.Now()
	}
	return
}
