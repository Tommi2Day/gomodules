// Package pwlib passwords encryption functions
package pwlib

// alternative: https://gist.github.com/wongoo/2b974a9594627114bea3e53c794980cd
import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
)

// GenRsaKey generate new key pair
func GenRsaKey(pubfilename string, privfilename string, password string) (publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey, err error) {
	bits := defaultRsaKeySize
	privateKey, err = rsa.GenerateKey(rand.Reader, bits)
	if err != nil || privateKey == nil {
		log.Debugf("generate Key failed: %s", err)
		return
	}
	publicKey = &privateKey.PublicKey

	// save to file if required
	if len(privfilename) > 0 {
		// Convert it to pem
		privbytes, _ := x509.MarshalPKCS8PrivateKey(privateKey)
		block := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privbytes,
		}

		// Encrypt the pem
		if password != "" {
			//nolint gosec
			block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(password), x509.PEMCipherAES256)
			if err != nil {
				log.Errorf("cannot encrypt private key %s", err)
				return
			}
		}
		// save it
		privatekeyPem := pem.EncodeToMemory(block)
		err = common.WriteStringToFile(privfilename, string(privatekeyPem))
		if err != nil {
			log.Errorf("cannot write %s: %s", privfilename, err)
			return
		}
		log.Debugf("private key written to %s", privfilename)
	}

	if len(pubfilename) > 0 {
		pubbytes, _ := x509.MarshalPKIXPublicKey(publicKey)
		block := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubbytes,
		}
		pubkeyPem := pem.EncodeToMemory(block)
		err = common.WriteStringToFile(pubfilename, string(pubkeyPem))
		if err != nil {
			log.Errorf("cannot write %s: %s", pubfilename, err)
			return
		}
		log.Debugf("public key written to %s", pubfilename)
	}
	log.Debug("Keys generated")
	return
}

// GetPrivateKeyFromFile  read private key from PEM encoded File and returns publicKey and private key objects
func GetPrivateKeyFromFile(privfilename string, rsaPrivateKeyPassword string) (publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey, err error) {
	var parsedKey interface{}
	var privPemBytes []byte

	log.Debugf("GetPrivateKeyFromFile entered for %s", privfilename)
	priv, err := common.ReadFileToString(privfilename)
	if err != nil {
		log.Debugf("cannot read %s: %s", privfilename, err)
		return
	}
	privPem, _ := pem.Decode([]byte(priv))
	if privPem == nil {
		log.Debugf("cannot decode pem in %s", privfilename)
		return
	}
	if privPem.Type != "RSA PRIVATE KEY" {
		log.Debugf("rsa private key is of the wrong type %s", privPem.Type)
		return
	}

	if rsaPrivateKeyPassword != "" {
		//nolint gosec
		privPemBytes, err = x509.DecryptPEMBlock(privPem, []byte(rsaPrivateKeyPassword))
		if err != nil {
			log.Debugf("rsa private password error:%s", err)
			return
		}
	} else {
		privPemBytes = privPem.Bytes
	}
	parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes)
	if err != nil {
		if strings.Contains(err.Error(), "use ParsePKCS1PrivateKey") {
			log.Debug("ParsePKCS8PrivateKey failed, trying ParsePKCS1PrivateKey")
			parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes)
		}
		if err != nil {
			log.Debugf("unable to parse RSA private key: %s", err)
			return
		}
	}
	privateKey = parsedKey.(*rsa.PrivateKey)
	if privateKey == nil {
		err = fmt.Errorf("unable to cast private key")
		log.Debugf("%s:", err)
		return
	}
	publicKey = &privateKey.PublicKey
	log.Debugf("Keys successfully loaded")
	return
}

// GetPublicKeyFromFile  read public key from PEM encoded File
func GetPublicKeyFromFile(publicKeyFile string) (publicKey *rsa.PublicKey, err error) {
	var parsedKey interface{}
	log.Debugf("load public key from %s", publicKeyFile)
	pub, err := common.ReadFileToString(publicKeyFile)
	if err != nil {
		log.Debugf("Cannot Read %s: %s", publicKeyFile, err)
		return
	}
	pubPem, _ := pem.Decode([]byte(pub))
	if pubPem == nil {
		log.Debugf("Cannot Decode %s", publicKeyFile)
		return
	}
	if pubPem.Type != "PUBLIC KEY" {
		log.Debugf("RSA public key is of the wrong type %s", pubPem.Type)
		return
	}

	parsedKey, err = x509.ParsePKIXPublicKey(pubPem.Bytes)
	if err != nil {
		log.Debugf("unable to parse RSA public key: %s", err)
		return
	}

	publicKey = parsedKey.(*rsa.PublicKey)
	if publicKey == nil {
		err = errors.New("unable to cast public key")
		log.Debugf("%s:", err)
	}
	log.Debugf("public key loaded successfully")
	return
}

// PubEncryptFileGo encrypts a file with public key with GO API
func PubEncryptFileGo(plainFile string, targetFile string, publicKeyFile string) (err error) {
	const rb = 16
	log.Debugf("Encrypt %s with public key %s", plainFile, publicKeyFile)
	publicKey, err := GetPublicKeyFromFile(publicKeyFile)
	if err != nil {
		return
	}
	sessionKey := make([]byte, rb)
	_, err = rand.Read(sessionKey)
	if err != nil {
		log.Debugf("Cannot generate session key:%s", err)
		return
	}
	plainData := ""
	plainData, err = common.ReadFileToString(plainFile)
	if err != nil {
		log.Debugf("Cannot read plaintext file %s:%s", plainFile, err)
		return
	}

	// sha1 for compatibility with python version
	hash := sha256.New()
	// oder rsa.EncryptPKCS1v15()
	encSessionKey, err := rsa.EncryptOAEP(hash, rand.Reader, publicKey, sessionKey, label)
	skl := len(encSessionKey)
	log.Debugf("Session key len: %d", skl)
	if err != nil {
		log.Error(err)
		return
	}
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		log.Debugf("Cannot create cipher: %s", err.Error())
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Debugf("Cannot create easgcm: %s", err.Error())
		return
	}
	ns := aesgcm.NonceSize()
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	// must be 12 for GCM
	nonce := make([]byte, ns)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		log.Debugf("Cannot create nounce: %s", err.Error())
		return
	}

	// do encryption and seal
	cipherdata := aesgcm.Seal(nil, nonce, []byte(plainData), nil)

	// encode all parts in base64
	bindata := bytes.Join([][]byte{encSessionKey, nonce, cipherdata}, []byte(""))
	b64 := base64.StdEncoding.EncodeToString(bindata)

	// write crypted output file
	err = common.WriteStringToFile(targetFile, b64)
	if err != nil {
		log.Debugf("Cannot write: %s", err.Error())
		return
	}
	return
}

// PrivateDecryptFileGo Decrypt a file with private key with GO API
func PrivateDecryptFileGo(cryptedfile string, privatekeyfile string, keypass string) (content string, err error) {
	log.Debugf("decrypt %s with private key %s", cryptedfile, privatekeyfile)
	data, err := common.ReadFileToString(cryptedfile)
	if err != nil {
		log.Debugf("Cannot Read file '%s': %s", cryptedfile, err)
		return
	}
	_, privkey, err := GetPrivateKeyFromFile(privatekeyfile, keypass)
	if err != nil {
		log.Debugf("Cannot read keys from '%s': %s", privatekeyfile, err)
		return
	}
	bindata, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Debugf("decode base64 for %s failed: %s", cryptedfile, err)
		return
	}
	s := 0
	e := privkey.Size()
	encSessionKey := bindata[s:e]

	hash := sha256.New()
	// oder rsa.EncryptPKCS1v15()
	sessionKey, err := rsa.DecryptOAEP(hash, rand.Reader, privkey, encSessionKey, label)
	if err != nil {
		log.Debugf("decode session key failed:%s", err)
		return
	}
	// sk := string(sessionKey)
	log.Debug("Session key decrypted")

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		log.Debugf("Cannot init cipher:%s", err)
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Debugf("Cannot setup AESGCM:%s", err)
		return
	}

	// split parts
	ns := aesgcm.NonceSize()
	s = e
	e = s + ns
	nonce := bindata[s:e]
	s = e
	cipherdata := bindata[s:]

	// do decrypt
	plaindata, err := aesgcm.Open(nil, nonce, cipherdata, nil)
	if err != nil {
		log.Debugf("Cannot decode crypted data:%s", err)
		return
	}
	// return content
	content = string(plaindata)
	log.Debug("Decoding successfully")
	return
}

// PrivateDecryptString Decrypt a string with private key
func PrivateDecryptString(crypted string, privatekeyfile string, keypass string) (plain string, err error) {
	// echo -n "$CRYPTED"|base64 -d  |openssl rsautl -decrypt -inkey ${PRIVATEKEYFILE} -passin pass:$KEYPASS
	log.Debugf("decrypt string with private key %s", privatekeyfile)
	_, privkey, err := GetPrivateKeyFromFile(privatekeyfile, keypass)
	if err != nil {
		log.Debugf("Cannot read keys from '%s': %s", privatekeyfile, err)
		return
	}
	b64dec, err := base64.StdEncoding.DecodeString(crypted)
	if err != nil {
		log.Debugf("decode base64 failed: %s", err)
		return
	}

	data, err := rsa.DecryptPKCS1v15(rand.Reader, privkey, b64dec)
	if err != nil {
		log.Debugf("decode session key failed:%s", err)
		return
	}
	plain = string(data)
	return
}

// PublicEncryptString  Encrypt a string with public key
func PublicEncryptString(plain string, publicKeyFile string) (crypted string, err error) {
	// echo -n "$plain"|openssl rsautl -encrypt -pkcs -inkey $PUBLICKEYFILE -pubin |base64
	log.Debugf("encrypt string with public key %s", publicKeyFile)
	pubkey, err := GetPublicKeyFromFile(publicKeyFile)
	if err != nil {
		log.Debugf("Cannot read keys from '%s': %s", publicKeyFile, err)
		return
	}

	data, err := rsa.EncryptPKCS1v15(rand.Reader, pubkey, []byte(plain))
	if err != nil {
		log.Debugf("decode session key failed:%s", err)
		return
	}
	crypted = base64.StdEncoding.EncodeToString(data)
	return
}
