package pwlib

import (
	"testing"
)

// TestTechProfile profile settings for technical users
var TestTechProfile = PasswordProfile{
	Length:      12,
	Upper:       1,
	Lower:       1,
	Digits:      1,
	Special:     1,
	FirstIsChar: true,
}

// TestUserProfile profile settings for personal users
var TestUserProfile = PasswordProfile{
	Length:      10,
	Upper:       1,
	Lower:       1,
	Digits:      1,
	Special:     0,
	FirstIsChar: true,
}

func TestDoPasswordCheck(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		valid   bool
		profile PasswordProfile
		special string
	}{
		{
			"NoCharacterAtAll",
			"",
			false,
			TestTechProfile,
			"",
		},
		{
			"JustEmptyStringAndWhitespace",
			" \n\t\r\v\f xxxx",
			false,
			TestTechProfile,
			"",
		},
		{
			"MixtureOfEmptyStringAndWhitespace",
			"U u\n1\t?\r1\v2\f34",
			false,
			TestTechProfile,
			"",
		},
		{
			"MissingUpperCaseString",
			"uu1?1234aaaa",
			false,
			TestTechProfile,
			"",
		},
		{
			"MissingLowerCaseString",
			"UU1?1234AAAA",
			false,
			TestTechProfile,
			"",
		},
		{
			"MissingNumber",
			"Uua?aaaaxxxx",
			false,
			TestTechProfile,
			"",
		},
		{
			"MissingSymbol",
			"Uu101234aaaa",
			false,
			TestTechProfile,
			"",
		},
		{
			"LessThanRequiredMinimumLength",
			"Uu1?123",
			false,
			TestTechProfile,
			"",
		},
		{
			"ValidPassword",
			"Uu1?1234aaaa",
			true,
			TestTechProfile,
			"",
		},
		{
			"InvalidSpecial",
			"Uu1?1234aaaa",
			false,
			TestTechProfile,
			"x!=@",
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			profile := c.profile
			special := c.special
			pps := PasswordProfileSet{
				Profile:      profile,
				SpecialChars: special,
			}
			pp, cs := pps.Load()
			if c.valid != DoPasswordCheck(c.pass, pp, cs) {
				t.Fatal("invalid password")
			}
		})
	}
}
