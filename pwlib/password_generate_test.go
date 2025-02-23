package pwlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenPasswordProfile(t *testing.T) {
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
			TestTechProfile.Length,
			TestTechProfile.Upper,
			TestTechProfile.Lower,
			TestTechProfile.Digits,
			TestTechProfile.Special,
			TestTechProfile.FirstIsChar,
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
			TestUserProfile.Length,
			TestUserProfile.Upper,
			TestUserProfile.Lower,
			TestUserProfile.Digits,
			TestUserProfile.Special,
			TestUserProfile.FirstIsChar,
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
			"User-16-2-2-2-2-1-no-charset",
			true,
			true,
			16,
			2,
			2,
			2,
			2,
			true,
			"",
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
			np := NewPasswordProfile(c.length, c.uc, c.lc, c.num, c.sp, c.first)
			pps := PasswordProfileSet{
				Profile:      np,
				SpecialChars: "",
			}
			pp, cs := pps.Load()
			newPassword, err := GenPasswordProfile(pps)
			t.Logf("generated Password: '%s'\n", newPassword)
			if err == nil {
				status := DoPasswordCheck(newPassword, pp, cs)
				if status != c.valid {
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

func TestGenPassword(t *testing.T) {
	// Returns valid password when called with empty profile string using DefaultPasswordProfile
	t.Run("TestGenPasswordWithEmptyProfile", func(t *testing.T) {
		password, err := GenPassword("")
		pps := PasswordProfileSet{Profile: DefaultPasswordProfile}
		pp, cs := pps.Load()
		require.NoError(t, err)
		require.NotEmpty(t, password)
		require.Len(t, password, pp.Length)

		// Verify password matches default profile requirements
		status := DoPasswordCheck(password, pp, cs)
		require.True(t, status)
	})

	// Handles invalid profile string format with appropriate error
	t.Run("TestGenPasswordWithInvalidProfile", func(t *testing.T) {
		invalidProfile := "invalid format"
		password, err := GenPassword(invalidProfile)
		require.Error(t, err)
		require.Empty(t, password)
		require.Contains(t, err.Error(), "profile string should have 6 space separated numbers")
	})

	// bGenerated password matches specified profile requirements for length and character types
	t.Run("TestGenPasswordWithValidProfile", func(t *testing.T) {
		profile := "16 1 1 1 1 1"
		password, err := GenPassword(profile)
		assert.NoError(t, err)
		assert.NotEmpty(t, password)
		assert.Equal(t, 16, len(password))
		pp, err := GetPasswordProfileFromString(profile)
		assert.NoError(t, err)
		cs := GetPasswordCharSet(DefaultSpecialChars)
		if err == nil {
			assert.True(t, DoPasswordCheck(password, pp, cs))
		}

		t.Logf("generated Password: '%s'\n", password)
	})

	// Correctly uses DefaultPasswordProfile when no profile is provided
	t.Run("TestGenPasswordWithDefaultProfile", func(t *testing.T) {
		password, err := GenPassword("")
		assert.NoError(t, err)
		assert.NotEmpty(t, password)
		cs := GetPasswordCharSet(DefaultSpecialChars)
		if err == nil {
			assert.True(t, DoPasswordCheck(password, DefaultPasswordProfile, cs))
		}
		assert.Equal(t, DefaultPasswordProfile.Length, len(password))
		assert.True(t, DoPasswordCheck(password, DefaultPasswordProfile, cs))
	})
}
