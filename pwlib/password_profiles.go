package pwlib

const (
	// UpperChar allowed charsets upper
	UpperChar = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// LowerChar allowed charsets lower
	LowerChar = "abcdefghijkmlnopqrstuvwxyz"
	// Digits allowed charsets digits
	Digits = "0123456789"
	// SpecialChar allowed charsets special
	SpecialChar = "!?()-_="
	// AllChars allowed charsets combined
	AllChars = UpperChar + LowerChar + Digits + SpecialChar
)

// PasswordProfile struct for password profile
type PasswordProfile struct {
	Length    int
	Upper     int
	Lower     int
	Digits    int
	Special   int
	Firstchar bool
}

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
