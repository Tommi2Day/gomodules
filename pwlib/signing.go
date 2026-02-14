package pwlib

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tommi2day/gomodules/common"
)

// SignFile signs the plaintext file
func (pc *PassConfig) SignFile() (err error) {
	plainFile := pc.PlainTextFile
	signatureFile := pc.SignatureFile
	privateKeyFile := pc.PrivateKeyFile
	keyPass := pc.KeyPass
	method := pc.Method
	keyID := pc.KMSKeyID

	log.Debugf("Sign %s with method %s", plainFile, method)
	switch method {
	case typeOpenssl, typeGO:
		err = SignFileSSL(plainFile, signatureFile, privateKeyFile, keyPass)
	case typeGPG:
		err = GPGSignFile(plainFile, signatureFile, privateKeyFile, keyPass)
	case typeKMS:
		var plain string
		plain, err = common.ReadFileToString(plainFile)
		if err != nil {
			return err
		}
		var svc = ConnectToKMS()
		var signature string
		signature, err = KMSSignString(svc, keyID, plain)
		if err == nil {
			err = common.WriteStringToFile(signatureFile, signature)
		}
	case typeAge:
		err = fmt.Errorf("signing not supported for age")
	default:
		err = fmt.Errorf("signing not supported for method %s", method)
	}

	if err != nil {
		log.Debug("signing failed")
		return
	}
	log.Debug("signing success")
	return
}

// VerifyFile verifies the signature of the plaintext file
func (pc *PassConfig) VerifyFile() (valid bool, err error) {
	plainFile := pc.PlainTextFile
	signatureFile := pc.SignatureFile
	pubKeyFile := pc.PubKeyFile
	method := pc.Method
	keyID := pc.KMSKeyID

	log.Debugf("Verify %s with method %s", plainFile, method)
	switch method {
	case typeOpenssl, typeGO:
		valid, err = VerifyFileSSL(plainFile, signatureFile, pubKeyFile)
	case typeGPG:
		valid, err = GPGVerifyFile(plainFile, signatureFile, pubKeyFile)
	case typeKMS:
		var plain string
		plain, err = common.ReadFileToString(plainFile)
		if err != nil {
			return false, err
		}
		var signature string
		signature, err = common.ReadFileToString(signatureFile)
		if err != nil {
			return false, err
		}
		var svc = ConnectToKMS()
		valid, err = KMSVerifyString(svc, keyID, plain, signature)
	case typeAge:
		err = fmt.Errorf("verification not supported for age")
	default:
		err = fmt.Errorf("verification not supported for method %s", method)
	}

	if err != nil {
		log.Debug("verification failed")
		return
	}
	log.Debugf("verification result: %v", valid)
	return
}
