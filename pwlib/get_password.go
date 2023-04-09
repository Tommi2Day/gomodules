package pwlib

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
)

// Methods all available methods for get_passwword
var Methods = []string{typeGO, typeOpenssl, typeEnc, typePlain}

// DecryptFile decripts an rsa protected file
func (pc *PassConfig) DecryptFile() (lines []string, err error) {
	cryptedfile := pc.CryptedFile
	privatekeyfile := pc.PrivateKeyFile
	keypass := pc.KeyPass
	sessionpassfile := pc.SessionPassFile
	passflag := "open"
	content := ""
	method := pc.Method
	var data []byte
	if len(keypass) > 0 {
		passflag = "Encypted"
	}
	log.Debugf("Decrypt data from %s with method %s(%s)", cryptedfile, method, passflag)

	switch method {
	case typeOpenssl:
		content, err = PrivateDecryptFileSSL(cryptedfile, privatekeyfile, keypass, sessionpassfile)
	case typeGO:
		content, err = PrivateDecryptFileGo(cryptedfile, privatekeyfile, keypass)
	case typeEnc:
		data, err = DecodeFile(cryptedfile)
		content = string(data)
	case typePlain:
		//nolint gosec
		data, err = os.ReadFile(cryptedfile)
		content = string(data)
	default:
		log.Fatalf("encryption method %s not known", method)
		os.Exit(1)
	}

	if err != nil {
		log.Debug("load data failed")
		return
	}
	content = strings.ReplaceAll(content, "\r", "")
	lines = strings.Split(content, "\n")
	log.Debug("load data success")
	return
}

// EncryptFile encrypt plain text to rsa protected file
func (pc *PassConfig) EncryptFile() (err error) {
	cryptedFile := pc.CryptedFile
	pubKeyFile := pc.PubKeyFile
	plaintextfile := pc.PlainTextFile
	sessionpassfile := pc.SessionPassFile
	method := pc.Method
	log.Debugf("Encrypt data from %s method %s", plaintextfile, method)
	switch method {
	case typeOpenssl:
		err = PubEncryptFileSSL(plaintextfile, cryptedFile, pubKeyFile, sessionpassfile)
	case typeGO:
		err = PubEncryptFileGo(plaintextfile, cryptedFile, pubKeyFile)
	case typeEnc:
		err = EncodeFile(plaintextfile, cryptedFile)
	case typePlain:
		// no need to do anything
		err = nil
	default:
		log.Fatalf("Enc method %s not known", method)
		os.Exit(1)
	}

	if err != nil {
		log.Debug("encryption data failed")
		return
	}
	log.Debug("encrytion data success")
	return
}

// ListPasswords printout list of pwcli
func (pc *PassConfig) ListPasswords() (lines []string, err error) {
	log.Debugf("ListPasswords entered")
	lines, err = pc.DecryptFile()
	if err != nil {
		log.Errorf("Decode Failed")
		return
	}
	return
}

// GetPassword ask System for data
func (pc *PassConfig) GetPassword(system string, account string) (password string, err error) {
	var lines []string
	log.Debugf("GetPassword for '%s'@'%s' entered", account, system)
	lines, err = pc.DecryptFile()
	if err != nil {
		return
	}
	found := false
	direct := false
	for _, line := range lines {
		if common.CheckSkip(line) {
			continue
		}
		fields := strings.SplitN(line, ":", 3)
		if len(fields) != 3 {
			log.Debugf("Skip incomplete record %s", line)
			continue
		}
		if system == fields[0] && account == fields[1] {
			log.Debug("Found direct match")
			if found {
				log.Debug("Overwrite previous default candidate")
			}
			found = true
			direct = true
			password = fields[2]
			break
		}
		if fields[0] == "!default" && account == fields[1] {
			password = fields[2]
			log.Debug("found new default match candidate")
			if found {
				log.Debug("Overwrite previous default candidate")
			}
			found = true
		}
	}
	// not found
	if !found {
		msg := fmt.Sprintf("no record found for '%s'@'%s'", account, system)
		log.Debug("GetPassword finished with no Match")
		err = errors.New(msg)
		return
	}

	// found
	if !direct {
		log.Debug("use default entry")
	}
	return
}
