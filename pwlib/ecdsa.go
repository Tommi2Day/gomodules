// Package pwlib passwords encryption functions
package pwlib

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
)

// GenEcdsaKey generate new ecdsa key pair
func GenEcdsaKey(pubfilename string, privfilename string, password string) (publicKey *ecdsa.PublicKey, privateKey *ecdsa.PrivateKey, err error) {
	curve := elliptic.P256()
	privateKey, err = ecdsa.GenerateKey(curve, rand.Reader)
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
			Type:  "EC PRIVATE KEY",
			Bytes: privbytes,
		}

		// Encrypt the pem
		if password != "" {
			//nolint:staticcheck
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

// GetEcdsaPrivateKeyFromFile read private key from PEM encoded File and returns publicKey and private key objects
func GetEcdsaPrivateKeyFromFile(privfilename string, password string) (publicKey *ecdsa.PublicKey, privateKey *ecdsa.PrivateKey, err error) {
	var parsedKey any
	var privPemBytes []byte

	log.Debugf("GetEcdsaPrivateKeyFromFile entered for %s", privfilename)
	priv, err := common.ReadFileToString(privfilename)
	if err != nil {
		log.Debugf("cannot read %s: %s", privfilename, err)
		return
	}
	privPem, _ := pem.Decode([]byte(priv))
	if privPem == nil {
		log.Debugf("cannot decode pem in %s", privfilename)
		err = fmt.Errorf("cannot decode pem in %s", privfilename)
		return
	}
	if privPem.Type != "EC PRIVATE KEY" && privPem.Type != "PRIVATE KEY" {
		log.Debugf("ecdsa private key is of the wrong type %s", privPem.Type)
		err = fmt.Errorf("ecdsa private key is of the wrong type %s", privPem.Type)
		return
	}

	if password != "" {
		//nolint:staticcheck
		privPemBytes, err = x509.DecryptPEMBlock(privPem, []byte(password))
		if err != nil {
			log.Debugf("ecdsa private password error:%s", err)
			return
		}
	} else {
		privPemBytes = privPem.Bytes
	}

	parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes)
	if err != nil {
		log.Debugf("ParsePKCS8PrivateKey failed: %s", err)
		// Try EC specific if PKCS8 fails
		parsedKey, err = x509.ParseECPrivateKey(privPemBytes)
		if err != nil {
			log.Debugf("unable to parse ECDSA private key: %s", err)
			return
		}
	}

	var ok bool
	privateKey, ok = parsedKey.(*ecdsa.PrivateKey)
	if !ok {
		err = errors.New("unable to cast private key")
		log.Debugf("%s:", err)
		return
	}
	publicKey = &privateKey.PublicKey
	log.Debugf("Keys successfully loaded")
	return
}

// GetEcdsaPublicKeyFromFile read public key from PEM encoded File
func GetEcdsaPublicKeyFromFile(publicKeyFile string) (publicKey *ecdsa.PublicKey, err error) {
	var parsedKey any
	log.Debugf("load public key from %s", publicKeyFile)
	pub, err := common.ReadFileToString(publicKeyFile)
	if err != nil {
		log.Debugf("Cannot Read %s: %s", publicKeyFile, err)
		return
	}
	pubPem, _ := pem.Decode([]byte(pub))
	if pubPem == nil {
		log.Debugf("Cannot Decode %s", publicKeyFile)
		err = fmt.Errorf("cannot decode pem in %s", publicKeyFile)
		return
	}
	if pubPem.Type != "PUBLIC KEY" {
		log.Debugf("ECDSA public key is of the wrong type %s", pubPem.Type)
		err = fmt.Errorf("ECDSA public key is of the wrong type %s", pubPem.Type)
		return
	}

	parsedKey, err = x509.ParsePKIXPublicKey(pubPem.Bytes)
	if err != nil {
		log.Debugf("unable to parse ECDSA public key: %s", err)
		return
	}

	var ok bool
	publicKey, ok = parsedKey.(*ecdsa.PublicKey)
	if !ok {
		err = errors.New("unable to cast public key")
		log.Debugf("%s:", err)
	}
	log.Debugf("public key loaded successfully")
	return
}

type ecdsaSignature struct {
	R, S *big.Int
}

// EcdsaSignString signs a string with private key
func EcdsaSignString(plain string, privatekeyfile string, keypass string) (signature string, err error) {
	log.Debugf("sign string with private key %s", privatekeyfile)
	_, privkey, err := GetEcdsaPrivateKeyFromFile(privatekeyfile, keypass)
	if err != nil {
		log.Debugf("Cannot read keys from '%s': %s", privatekeyfile, err)
		return
	}

	hash := sha256.Sum256([]byte(plain))
	r, s, err := ecdsa.Sign(rand.Reader, privkey, hash[:])
	if err != nil {
		log.Debugf("sign failed: %s", err)
		return
	}

	sig, err := asn1.Marshal(ecdsaSignature{r, s})
	if err != nil {
		log.Debugf("marshal signature failed: %s", err)
		return
	}

	signature = base64.StdEncoding.EncodeToString(sig)
	return
}

// EcdsaVerifyString verifies a string signature with public key
func EcdsaVerifyString(plain string, signature string, publicKeyFile string) (valid bool, err error) {
	log.Debugf("verify string with public key %s", publicKeyFile)
	pubkey, err := GetEcdsaPublicKeyFromFile(publicKeyFile)
	if err != nil {
		log.Debugf("Cannot read keys from '%s': %s", publicKeyFile, err)
		return
	}

	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		log.Debugf("decode signature failed: %s", err)
		return
	}

	var ecdsaSig ecdsaSignature
	_, err = asn1.Unmarshal(sig, &ecdsaSig)
	if err != nil {
		log.Debugf("unmarshal signature failed: %s", err)
		return
	}

	hash := sha256.Sum256([]byte(plain))
	valid = ecdsa.Verify(pubkey, hash[:], ecdsaSig.R, ecdsaSig.S)
	return
}
