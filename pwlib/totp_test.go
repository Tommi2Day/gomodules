package pwlib

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

const wrong = "xxx"
const ok = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

// TestGetOutput Test GetOutput output should return expected value
func TestGetOTP(t *testing.T) {
	t.Run("invalid secret", func(t *testing.T) {
		_, err := GetOtp(wrong)
		assert.Errorf(t, err, "Invalid secret given, but claims ok")
	})
	t.Run("valid secret", func(t *testing.T) {
		val, err := GetOtp(ok)
		assert.NoErrorf(t, err, "valid secret given, but claims failed")
		assert.NotEmpty(t, val, "valid secret given, answer empty")
		assert.Lenf(t, val, 6, "should have exact 6 char")
		assert.Regexpf(t, regexp.MustCompile(`^\d+$`), val, "should only digits")
		t.Logf("OTP: %s", val)
	})
}
