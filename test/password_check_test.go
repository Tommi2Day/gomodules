package test

import (
	"github.com/tommi2day/gomodules/pwlib"
	"testing"
)

func TestDoPasswordCheck(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		valid   bool
		profile pwlib.PasswordProfile
	}{
		{
			"NoCharacterAtAll",
			"",
			false,
			pwlib.TechProfile,
		},
		{
			"JustEmptyStringAndWhitespace",
			" \n\t\r\v\f xxxx",
			false,
			pwlib.TechProfile,
		},
		{
			"MixtureOfEmptyStringAndWhitespace",
			"U u\n1\t?\r1\v2\f34",
			false,
			pwlib.TechProfile,
		},
		{
			"MissingUpperCaseString",
			"uu1?1234aaaa",
			false,
			pwlib.TechProfile,
		},
		{
			"MissingLowerCaseString",
			"UU1?1234AAAA",
			false,
			pwlib.TechProfile,
		},
		{
			"MissingNumber",
			"Uua?aaaaxxxx",
			false,
			pwlib.TechProfile,
		},
		{
			"MissingSymbol",
			"Uu101234aaaa",
			false,
			pwlib.TechProfile,
		},
		{
			"LessThanRequiredMinimumLength",
			"Uu1?123",
			false,
			pwlib.TechProfile,
		},
		{
			"ValidPassword",
			"Uu1?1234aaaa",
			true,
			pwlib.TechProfile,
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			profile := c.profile
			if c.valid != pwlib.DoPasswordCheck(c.pass, profile.Length, profile.Upper, profile.Lower, profile.Digits, profile.Special, profile.Firstchar, "") {
				t.Fatal("invalid password")
			}
		})
	}
}
