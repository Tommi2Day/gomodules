package pwlib

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/html/charset"
	"gopkg.in/yaml.v3"

	log "github.com/sirupsen/logrus"
)

const (
	// UpperChar allowed charsets upper
	UpperChar = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// LowerChar allowed charsets lower
	LowerChar = "abcdefghijkmlnopqrstuvwxyz"
	// Digits allowed charsets digits
	Digits = "0123456789"
	// DefaultSpecialChars allowed charsets special
	DefaultSpecialChars = "!?#()$-_="
	// AllChars allowed charsets combined
	AllChars = UpperChar + LowerChar + Digits + DefaultSpecialChars
)

// PasswordCharset defines the allowed characters to choose
type PasswordCharset struct {
	// UpperChar allowed charsets upper
	UpperChar string
	// LowerChar allowed charsets lower
	LowerChar string
	// Digits allowed charsets digits
	Digits string
	// SpecialChar allowed charsets special
	SpecialChar string
	// AllChars allowed charsets combined
	AllChars string
}

// PasswordProfile struct for password profile
type PasswordProfile struct {
	Length      int  `yaml:"length" json:"length"`
	Upper       int  `yaml:"upper" json:"upper"`
	Lower       int  `yaml:"lower" json:"lower"`
	Digits      int  `yaml:"digits" json:"digits"`
	Special     int  `yaml:"specials,omitempty" json:"specials,omitempty"`
	FirstIsChar bool `yaml:"first_is_char" json:"first_is_char"`
}

// PasswordProfileSet defines a structure that combines a password profile and optional special character overrides.
type PasswordProfileSet struct {
	Profile      PasswordProfile `yaml:"profile" json:"profile"`
	SpecialChars string          `yaml:"special_chars,omitempty" json:"special_chars,omitempty"`
}

// PasswordProfileSets is a mapping between profile names and their associated PasswordProfileSet configurations.
type PasswordProfileSets map[string]PasswordProfileSet

// DefaultPasswordProfile defines the standard password profile configuration with default character requirements and length.
var DefaultPasswordProfile = PasswordProfile{
	Length:      16,
	Upper:       1,
	Lower:       1,
	Digits:      1,
	Special:     1,
	FirstIsChar: true,
}

// DefaultPasswordProfileSets is a pre-defined collection of password profiles with default settings and special characters.
var DefaultPasswordProfileSets = PasswordProfileSets{
	"default": PasswordProfileSet{
		Profile:      DefaultPasswordProfile,
		SpecialChars: DefaultSpecialChars,
	},
}

// LoadPasswordProfileSets parses input data string (JSON, XML, or YAML) into PasswordProfileSets and returns it.
// Returns an error if no data is provided or if parsing fails.
func LoadPasswordProfileSets(data string) (r PasswordProfileSets, err error) {
	data = strings.TrimLeft(data, "\n\t ")
	r = nil
	ps := PasswordProfileSets{}
	c := []byte(data)
	if len(c) == 0 {
		err = fmt.Errorf("no data to parse")
		return
	}
	switch {
	case strings.HasPrefix(data, "{"):
		err = json.Unmarshal(c, &ps)
	case strings.HasPrefix(data, "<"):
		decoder := xml.NewDecoder(bytes.NewReader(c))
		decoder.CharsetReader = charset.NewReaderLabel
		err = decoder.Decode(&ps)
	default:
		err = yaml.Unmarshal(c, &ps)
	}
	if err != nil {
		err = fmt.Errorf("error parsing input: %v", err)
		return
	}
	if len(ps) > 0 {
		r = ps
	}
	return
}

// String returns the YAML-encoded representation of the PasswordProfileSet as a string or an empty string on error.
func (pps PasswordProfileSet) String() string {
	b, err := yaml.Marshal(pps)
	if err != nil {
		return ""
	}
	return string(b)
}

// NewPasswordProfile creates a new PasswordProfile struct with specified settings for password generation.
func NewPasswordProfile(length int, upper int, lower int, digits int, special int, firstIsChar bool) PasswordProfile {
	return PasswordProfile{
		Length:      length,
		Upper:       upper,
		Lower:       lower,
		Digits:      digits,
		Special:     special,
		FirstIsChar: firstIsChar,
	}
}

// GetPasswordProfileFromString generates a PasswordProfile based on a given profile string or defaults to a predefined profile.
// Returns an error if the input profile string is invalid or does not meet the format requirements.
func GetPasswordProfileFromString(profile string) (pp PasswordProfile, err error) {
	if len(profile) == 0 {
		pp = DefaultPasswordProfile
		log.Info("Choose default password profile ")
		return
	}
	custom := strings.Split(profile, " ")
	if len(custom) < 6 {
		err = fmt.Errorf("profile string should have 6 space separated numbers <length> <upper chars> <lower chars> <digits> <special chars> <do firstchar check(0/1)>")
		return
	}
	pp.Length, err = strconv.Atoi(custom[0])
	if err == nil {
		pp.Upper, err = strconv.Atoi(custom[1])
	}
	if err == nil {
		pp.Lower, err = strconv.Atoi(custom[2])
	}
	if err == nil {
		pp.Digits, err = strconv.Atoi(custom[3])
	}
	if err == nil {
		pp.Special, err = strconv.Atoi(custom[4])
	}
	if err == nil {
		f := custom[5]
		pp.FirstIsChar = f == "1"
	}
	return
}

// Load initializes a password profile by setting default special characters if none are provided and updates global configurations.
func (pps PasswordProfileSet) Load() (profile PasswordProfile, profileCharset PasswordCharset) {
	if pps.SpecialChars == "" && pps.Profile.Special > 0 {
		log.Info("No special characters defined but specials requested, using default")
		pps.SpecialChars = DefaultSpecialChars
	}
	profileCharset = GetPasswordCharSet(pps.SpecialChars)
	profile = pps.Profile
	return
}

// ActivateProfile sets the active password profile by its name, initializes it, and updates global configuration variables.
// Returns an error if the profile name is not found or if no profiles are defined.
func (ps PasswordProfileSets) ActivateProfile(profileName string) (profile PasswordProfile, profileCharset PasswordCharset, err error) {
	var pps PasswordProfileSet
	if len(ps) == 0 {
		err = fmt.Errorf("no profiles defined")
		return
	}
	ok := false
	if pps, ok = ps[profileName]; !ok {
		err = fmt.Errorf("profile %s not found", profileName)
		return
	}
	profile, profileCharset = pps.Load()
	return
}

// GetPasswordCharSet generates a PasswordCharset containing specified character groups, including custom special characters.
func GetPasswordCharSet(specialChars string) PasswordCharset {
	all := UpperChar + LowerChar + Digits + specialChars
	c := PasswordCharset{UpperChar, LowerChar, Digits, specialChars, all}
	return c
}

// init initializes the default password profile set by activating the "default" profile and updating global configurations.
func init() {
	_, _, _ = DefaultPasswordProfileSets.ActivateProfile("default")
}
