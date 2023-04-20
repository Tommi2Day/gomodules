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
	Length    int
	Upper     int
	Lower     int
	Digits    int
	Special   int
	Firstchar bool
}

var charset = PasswordCharset{UpperChar, LowerChar, Digits, SpecialChar, UpperChar + LowerChar + Digits + SpecialChar}
