package pwlib

// https://ferdinand-neman.medium.com/ssha-password-hash-with-golang-7d79d792bd3d
import (
	"bytes"
	//nolint gosec
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"
)

// SSHASaltLen is the length of the salt for SSHA
const SSHASaltLen = 4

// SSHAPrefix is the prefix for SSHA
const SSHAPrefix = "{SSHA}"

// SSHEncoder is the encoder object for SSHA
type SSHAEncoder struct {
}

// Encode encodes the []byte of raw password
func (enc SSHAEncoder) Encode(rawPassPhrase []byte, prefix string) ([]byte, error) {
	salt, err := makeSSHASalt()
	if err != nil {
		return []byte{}, err
	}
	hash := makeSSHAHash(rawPassPhrase, salt)
	b64 := base64.StdEncoding.EncodeToString(hash)
	return []byte(fmt.Sprintf("%s%s", prefix, b64)), nil
}

// Matches matches the encoded password and the raw password
func (enc SSHAEncoder) Matches(encodedPassPhrase, rawPassPhrase []byte) bool {
	// strip the {SSHA}
	eppS := string(encodedPassPhrase)
	if strings.HasPrefix(string(encodedPassPhrase), SSHAPrefix) {
		eppS = string(encodedPassPhrase)[6:]
	}
	hash, err := base64.StdEncoding.DecodeString(eppS)
	if err != nil {
		return false
	}
	salt := hash[len(hash)-4:]
	//nolint gosec
	sha := sha1.New()
	_, _ = sha.Write(rawPassPhrase)
	_, _ = sha.Write(salt)
	sum := sha.Sum(nil)

	// compare without the last 4 bytes of the hash with the calculated hash
	return bytes.Equal(sum, hash[:len(hash)-SSHASaltLen])
}

// makeSSHASalt make 4Byte salt for SSHA hashing
func makeSSHASalt() (salt []byte, err error) {
	salt, err = makeSalt(SSHASaltLen)
	return
}

// makeSSHAHash make hasing using SHA-1 with salt. This is not the final output though. You need to append {SSHA} string with base64 of this hash.
func makeSSHAHash(passphrase, salt []byte) []byte {
	//nolint gosec
	sha := sha1.New()
	_, _ = sha.Write(passphrase)
	_, _ = sha.Write(salt)

	h := sha.Sum(nil)
	return append(h, salt...)
}
