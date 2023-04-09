package pwlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// nolint gocyclo
func TestDoPasswordCheck(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		valid   bool
		profile PasswordProfile
	}{
		{
			"NoCharacterAtAll",
			"",
			false,
			TechProfile,
		},
		{
			"JustEmptyStringAndWhitespace",
			" \n\t\r\v\f xxxx",
			false,
			TechProfile,
		},
		{
			"MixtureOfEmptyStringAndWhitespace",
			"U u\n1\t?\r1\v2\f34",
			false,
			TechProfile,
		},
		{
			"MissingUpperCaseString",
			"uu1?1234aaaa",
			false,
			TechProfile,
		},
		{
			"MissingLowerCaseString",
			"UU1?1234AAAA",
			false,
			TechProfile,
		},
		{
			"MissingNumber",
			"Uua?aaaaxxxx",
			false,
			TechProfile,
		},
		{
			"MissingSymbol",
			"Uu101234aaaa",
			false,
			TechProfile,
		},
		{
			"LessThanRequiredMinimumLength",
			"Uu1?123",
			false,
			TechProfile,
		},
		{
			"ValidPassword",
			"Uu1?1234aaaa",
			true,
			TechProfile,
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			profile := c.profile
			actual := DoPasswordCheck(c.pass, profile.Length, profile.Upper, profile.Lower, profile.Digits, profile.Special, profile.Firstchar, "")
			assert.Equalf(t, c.valid, actual, "Check %s not as expected", c.name)
		})
	}
}
