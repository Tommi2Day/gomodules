package pwlib

import (
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
			TechProfile.Length,
			TechProfile.Upper,
			TechProfile.Lower,
			TechProfile.Digits,
			TechProfile.Special,
			TechProfile.Firstchar,
			AllChars,
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
			AllChars,
		},
		{
			"Personal User",
			true,
			true,
			UserProfile.Length,
			UserProfile.Upper,
			UserProfile.Lower,
			UserProfile.Digits,
			UserProfile.Special,
			UserProfile.Firstchar,
			AllChars,
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
			AllChars,
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
			UpperChar + Digits,
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			newPassword, err := GenPassword(c.length, c.uc, c.lc, c.num, c.sp, c.first)
			t.Logf("generated Password: '%s'\n", newPassword)
			if err == nil {
				ok := DoPasswordCheck(newPassword, c.length, c.uc, c.lc, c.num, c.sp, c.first, c.chars)
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
