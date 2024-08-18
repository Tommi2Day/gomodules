package pwlib

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/tommi2day/gomodules/common"

	openssl "github.com/Luzifer/go-openssl/v4"
	log "github.com/sirupsen/logrus"
)

// PrivateDecryptFileSSL Decrypt a file with private key with openssl API
func PrivateDecryptFileSSL(cryptedFile string, privateKeyFile string, keyPass string, sessionPassFile string) (content string, err error) {
	log.Debugf("decrypt %s with private key %s in OpenSSL format", cryptedFile, privateKeyFile)
	cryptedkey := ""

	data, err := common.ReadFileToString(cryptedFile)
	if err != nil {
		log.Debugf("Cannot Read file '%s': %s", cryptedFile, err)
		return
	}
	if len(sessionPassFile) > 0 {
		cryptedkey, err = common.ReadFileToString(sessionPassFile)
		if err != nil {
			log.Debugf("Cannot Read file '%s': %s", sessionPassFile, err)
			return
		}
	}
	crypted := []byte(data)
	sessionKey, err := PrivateDecryptString(cryptedkey, privateKeyFile, keyPass)
	if err != nil {
		log.Debugf("Cannot decrypt Session Key from '%s': %s", sessionPassFile, err)
		return
	}
	// OPENSSL enc -d -aes-256-cbc -md sha256 -base64 -in $SOURCE -pass pass:$PASSPHRASE
	o := openssl.New()
	decoded, err := o.DecryptBytes(sessionKey, crypted, SSLDigest)
	if err != nil {
		log.Debugf("Cannot decrypt data from '%s': %s", cryptedFile, err)
		return
	}
	content = string(decoded)
	return
}

// PubEncryptFileSSL encrypts a file with public key with openssl API
func PubEncryptFileSSL(plainFile string, targetFile string, publicKeyFile string, sessionPassFile string) (err error) {
	const rb = 16
	log.Debugf("Encrypt %s with public key %s in OpenSSL format", plainFile, publicKeyFile)
	random := make([]byte, rb)
	_, err = rand.Read(random)
	if err != nil {
		log.Debugf("Cannot generate session key:%s", err)
		return
	}
	sessionKey := base64.StdEncoding.EncodeToString(random)
	crypted, err := PublicEncryptString(sessionKey, publicKeyFile)
	if err != nil {
		log.Errorf("Encrypting Keyfile failed: %v", err)
	}

	if len(sessionPassFile) > 0 {
		err = common.WriteStringToFile(sessionPassFile, crypted)
		if err != nil {
			log.Errorf("Cannot write session Key file %s:%v", sessionPassFile, err)
		}
	}

	//nolint gosec
	plainData, err := common.ReadFileToString(plainFile)
	if err != nil {
		log.Debugf("Cannot read plaintext file %s:%s", plainFile, err)
		return
	}

	o := openssl.New()
	// openssl enc -e -aes-256-cbc -md sha246 -base64 -in $SOURCE -out $TARGET -pass pass:$PASSPHRASE
	encrypted, err := o.EncryptBytes(sessionKey, []byte(plainData), SSLDigest)
	if err != nil {
		log.Errorf("cannot encrypt plaintext file %s:%s", plainFile, err)
		return
	}

	// write crypted output file
	err = common.WriteStringToFile(targetFile, string(encrypted))
	if err != nil {
		log.Errorf("Cannot write: %s", err.Error())
		return
	}
	return
}
