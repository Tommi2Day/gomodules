package pwlib

import (
	"testing"
)

// TechProfile profile settings for technical users
var TechProfile = PasswordProfile{
	Length:    12,
	Upper:     1,
	Lower:     1,
	Digits:    1,
	Special:   1,
	Firstchar: true,
}

// UserProfile profile settings for personal users
var UserProfile = PasswordProfile{
	Length:    10,
	Upper:     1,
	Lower:     1,
	Digits:    1,
	Special:   0,
	Firstchar: true,
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
			TechProfile,
			"",
		},
		{
			"JustEmptyStringAndWhitespace",
			" \n\t\r\v\f xxxx",
			false,
			TechProfile,
			"",
		},
		{
			"MixtureOfEmptyStringAndWhitespace",
			"U u\n1\t?\r1\v2\f34",
			false,
			TechProfile,
			"",
		},
		{
			"MissingUpperCaseString",
			"uu1?1234aaaa",
			false,
			TechProfile,
			"",
		},
		{
			"MissingLowerCaseString",
			"UU1?1234AAAA",
			false,
			TechProfile,
			"",
		},
		{
			"MissingNumber",
			"Uua?aaaaxxxx",
			false,
			TechProfile,
			"",
		},
		{
			"MissingSymbol",
			"Uu101234aaaa",
			false,
			TechProfile,
			"",
		},
		{
			"LessThanRequiredMinimumLength",
			"Uu1?123",
			false,
			TechProfile,
			"",
		},
		{
			"ValidPassword",
			"Uu1?1234aaaa",
			true,
			TechProfile,
			"",
		},
		{
			"InvalidSpecial",
			"Uu1?1234aaaa",
			false,
			TechProfile,
			"x!=@",
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			profile := c.profile
			special := c.special
			if special != "" {
				SetSpecialChars(special)
			}
			if c.valid != DoPasswordCheck(c.pass, profile.Length, profile.Upper, profile.Lower, profile.Digits, profile.Special, profile.Firstchar, charset.AllChars) {
				t.Fatal("invalid password")
			}
		})
	}
}
