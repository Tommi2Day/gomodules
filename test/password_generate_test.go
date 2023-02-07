package test

import (
	"github.com/tommi2day/gomodules/pwlib"
	"testing"
)

func TestGenPassword(t *testing.T) {
	tests := []struct {
		name     string
		genValid bool
		valid    bool
		length   int
		uc       int
		lc       int
		num      int
		sp       int
		first    bool
		chars    string
	}{
		{
			"Default Techuser",
			true,
			true,
			pwlib.TechProfile.Length,
			pwlib.TechProfile.Upper,
			pwlib.TechProfile.Lower,
			pwlib.TechProfile.Digits,
			pwlib.TechProfile.Special,
			pwlib.TechProfile.Firstchar,
			pwlib.AllChars,
		},
		{
			"InvalidGenZeroLength",
			false,
			false,
			0,
			0,
			0,
			0,
			0,
			false,
			pwlib.AllChars,
		},
		{
			"Personal User",
			true,
			true,
			pwlib.UserProfile.Length,
			pwlib.UserProfile.Upper,
			pwlib.UserProfile.Lower,
			pwlib.UserProfile.Digits,
			pwlib.UserProfile.Special,
			pwlib.UserProfile.Firstchar,
			pwlib.AllChars,
		},
		{
			"User-16-2-2-2-2-1",
			true,
			true,
			16,
			2,
			2,
			2,
			2,
			true,
			pwlib.AllChars,
		},
		{
			"Only-8-UpperAndDigits",
			true,
			true,
			8,
			1,
			0,
			1,
			0,
			false,
			pwlib.UpperChar + pwlib.Digits,
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			newPassword, err := pwlib.GenPassword(c.length, c.uc, c.lc, c.num, c.sp, c.first)
			t.Logf("generated Password: '%s'\n", newPassword)
			if err == nil {
				ok := pwlib.DoPasswordCheck(newPassword, c.length, c.uc, c.lc, c.num, c.sp, c.first, c.chars)
				if ok != c.valid {
					t.Fatalf("invalid password '%s'", newPassword)
				}
			} else
			// generation failed
			if c.genValid {
				t.Fatalf("Password Generation failed: %s", err.Error())
			}
		})
	}
}
