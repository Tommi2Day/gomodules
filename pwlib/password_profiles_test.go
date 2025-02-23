package pwlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPasswordProfiles(t *testing.T) {
	t.Run("TestSetPasswordProfileWithValidString", testSetPasswordProfileWithValidString)
	t.Run("TestSetPasswordProfileWithInvalidParamCount", testSetPasswordProfileWithInvalidParamCount)
	t.Run("TestEmptyProfileStringReturnsDefault", testEmptyProfileStringReturnsDefault)
	t.Run("TestLoadPasswordProfilesFromStringWithValidJSON", testLoadPasswordProfilesFromStringWithValidJSON)
	t.Run("TestLoadPasswordProfilesFromStringWithValidYAML", testLoadPasswordProfilesFromStringWithValidYAML)
	t.Run("TestLoadPasswordProfilesFromStringWithEmptyInput", testLoadPasswordProfilesFromStringWithEmptyInput)
	t.Run("TestLoadPasswordProfilesFromStringWithInvalidJsonInput", testLoadPasswordProfilesFromStringWithInvalidJSONInput)
	t.Run("TestLoadPasswordProfilesFromStringWithInvalidYamlInput", testLoadPasswordProfilesFromStringWithInvalidYamlInput)
	t.Run("TestActivateProfileWithDefaultSpecialChars", testActivateProfileWithDefaultSpecialChars)
	t.Run("TestActivateNonExistentProfile", testActivateNonExistentProfile)
	t.Run("TestLoadProfileWithDefaultSpecialChars", testLoadProfileWithDefaultSpecialChars)
	t.Run("TestNewPasswordProfileWithValidParameters", testNewPasswordProfileWithValidParameters)
	t.Run("TestNewPasswordProfileWithNegativeLength", testNewPasswordProfileWithNegativeLength)
	t.Run("TestNewPasswordProfileFirstIsCharTrue", testNewPasswordProfileFirstIsCharTrue)
	t.Run("TestNewPasswordProfileFirstIsCharFalse", testNewPasswordProfileFirstIsCharFalse)
	t.Run("TestNewPasswordProfileAllZero", testNewPasswordProfileAllZero)
}

func testSetPasswordProfileWithValidString(t *testing.T) {
	profileStr := "16 2 3 2 1 1"
	profile, err := GetPasswordProfileFromString(profileStr)
	assert.NoError(t, err)
	assert.Equal(t, 16, profile.Length)
	assert.Equal(t, 2, profile.Upper)
	assert.Equal(t, 3, profile.Lower)
	assert.Equal(t, 2, profile.Digits)
	assert.Equal(t, 1, profile.Special)
	assert.True(t, profile.FirstIsChar)
}

func testSetPasswordProfileWithInvalidParamCount(t *testing.T) {
	profileStr := "16 2 3 2"
	_, err := GetPasswordProfileFromString(profileStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profile string should have 6 space separated numbers")
}

func testEmptyProfileStringReturnsDefault(t *testing.T) {
	expectedProfile := DefaultPasswordProfile
	profile, err := GetPasswordProfileFromString("")
	assert.NoError(t, err)
	assert.Equal(t, expectedProfile, profile)
}

func testLoadPasswordProfilesFromStringWithValidJSON(t *testing.T) {
	input := `{
		"profile1": {
			"profile": {
				"length": 12,
				"upper": 2,
				"lower": 2,
				"digits": 2,
				"specials": 2
			},
            "special_chars": "!@#$"
        }
}`
	profiles, err := LoadPasswordProfileSets(input)
	assert.NoError(t, err)
	assert.NotNil(t, profiles)
	assert.Equal(t, 1, len(profiles))
	assert.Contains(t, profiles, "profile1")
	if pp, success := profiles["profile1"]; success {
		assert.Equal(t, 12, pp.Profile.Length)
		assert.Equal(t, "!@#$", pp.SpecialChars)
		assert.Equal(t, 2, pp.Profile.Upper)
		assert.Equal(t, 2, pp.Profile.Lower)
		assert.Equal(t, 2, pp.Profile.Digits)
		assert.Equal(t, 2, pp.Profile.Special)
		assert.False(t, pp.Profile.FirstIsChar)
	}
}

func testLoadPasswordProfilesFromStringWithValidYAML(t *testing.T) {
	input := `
	profile2: 
      # Length Upper Lower Digits Specials FirstIsChar
      profile:
        length: 16
        upper: 1 
        lower: 1 
        digits: 1 
        specials: 0
        first_is_char: true
      special_chars: "!!@#$%^&*()_+-="
 `
	profiles, err := LoadPasswordProfileSets(input)
	assert.NoError(t, err)
	assert.NotNil(t, profiles)
	assert.Equal(t, 1, len(profiles))
	assert.Contains(t, profiles, "profile2")
	if pp, success := profiles["profile2"]; success {
		assert.Equal(t, 16, pp.Profile.Length)
		assert.Equal(t, "!!@#$%^&*()_+-=", pp.SpecialChars)
		assert.Equal(t, 1, pp.Profile.Upper)
		assert.Equal(t, 1, pp.Profile.Lower)
		assert.Equal(t, 1, pp.Profile.Digits)
		assert.Equal(t, 0, pp.Profile.Special)
		assert.True(t, pp.Profile.FirstIsChar)
	}
}

func testLoadPasswordProfilesFromStringWithEmptyInput(t *testing.T) {
	input := ""
	profiles, err := LoadPasswordProfileSets(input)
	assert.Error(t, err)
	assert.Equal(t, "no data to parse", err.Error())
	assert.Nil(t, profiles)
}

func testLoadPasswordProfilesFromStringWithInvalidJSONInput(t *testing.T) {
	input := `{ "Test": "test" }`
	profiles, err := LoadPasswordProfileSets(input)
	assert.Error(t, err)
	assert.Equal(t, 0, len(profiles))
	assert.Contains(t, err.Error(), "error parsing input")
}

func testLoadPasswordProfilesFromStringWithInvalidYamlInput(t *testing.T) {
	input := "Test: test "
	profiles, err := LoadPasswordProfileSets(input)
	assert.Error(t, err)
	assert.Equal(t, 0, len(profiles))
	assert.Contains(t, err.Error(), "error parsing input")
}

func testActivateProfileWithDefaultSpecialChars(t *testing.T) {
	ps := PasswordProfileSets{
		"test": PasswordProfileSet{
			Profile: PasswordProfile{
				Length:      16,
				Upper:       1,
				Lower:       1,
				Digits:      1,
				Special:     1,
				FirstIsChar: true,
			},
		},
	}
	pp, cs, err := ps.ActivateProfile("test")
	assert.NoError(t, err)
	assert.Equal(t, DefaultSpecialChars, cs.SpecialChar)
	expectedAllChars := UpperChar + LowerChar + Digits + DefaultSpecialChars
	assert.Equal(t, expectedAllChars, cs.AllChars)
	assert.Equal(t, 16, pp.Length)
}

func testActivateNonExistentProfile(t *testing.T) {
	ps := PasswordProfileSets{
		"test": PasswordProfileSet{
			Profile: PasswordProfile{
				Length:      16,
				Upper:       1,
				Lower:       1,
				Digits:      1,
				Special:     1,
				FirstIsChar: true,
			},
		},
	}
	_, _, err := ps.ActivateProfile("nonexistent")
	assert.Error(t, err)
	assert.Equal(t, "profile nonexistent not found", err.Error())
}

func testLoadProfileWithDefaultSpecialChars(t *testing.T) {
	profile := PasswordProfile{
		Length:      16,
		Upper:       1,
		Lower:       1,
		Digits:      1,
		Special:     1,
		FirstIsChar: true,
	}
	profileSet := PasswordProfileSet{
		Profile:      profile,
		SpecialChars: "",
	}
	pp, cs := profileSet.Load()
	assert.Equal(t, DefaultSpecialChars, cs.SpecialChar)
	expectedAllChars := UpperChar + LowerChar + Digits + DefaultSpecialChars
	assert.Equal(t, expectedAllChars, cs.AllChars)
	assert.Equal(t, profile, pp)
}

func testNewPasswordProfileWithValidParameters(t *testing.T) {
	profile := NewPasswordProfile(16, 2, 3, 4, 1, true)
	assert.Equal(t, 16, profile.Length)
	assert.Equal(t, 2, profile.Upper)
	assert.Equal(t, 3, profile.Lower)
	assert.Equal(t, 4, profile.Digits)
	assert.Equal(t, 1, profile.Special)
	assert.True(t, profile.FirstIsChar)
}

func testNewPasswordProfileWithNegativeLength(t *testing.T) {
	profile := NewPasswordProfile(-5, 1, 1, 1, 1, true)
	assert.Equal(t, -5, profile.Length)
	assert.Equal(t, 1, profile.Upper)
	assert.Equal(t, 1, profile.Lower)
	assert.Equal(t, 1, profile.Digits)
	assert.Equal(t, 1, profile.Special)
	assert.True(t, profile.FirstIsChar)
}

func testNewPasswordProfileFirstIsCharTrue(t *testing.T) {
	profile := NewPasswordProfile(10, 2, 3, 2, 1, true)
	assert.True(t, profile.FirstIsChar)
}

func testNewPasswordProfileFirstIsCharFalse(t *testing.T) {
	profile := NewPasswordProfile(10, 2, 3, 2, 1, false)
	assert.False(t, profile.FirstIsChar)
}

func testNewPasswordProfileAllZero(t *testing.T) {
	profile := NewPasswordProfile(0, 0, 0, 0, 0, false)
	assert.Equal(t, 0, profile.Length)
	assert.Equal(t, 0, profile.Upper)
	assert.Equal(t, 0, profile.Lower)
	assert.Equal(t, 0, profile.Digits)
	assert.Equal(t, 0, profile.Special)
	assert.False(t, profile.FirstIsChar)
}
