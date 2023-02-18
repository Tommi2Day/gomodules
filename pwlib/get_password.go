package pwlib

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
)

// DecryptFile decripts an rsa protected file
func DecryptFile() (lines []string, err error) {
	cryptedfile := PwConfig.CryptedFile
	privatekeyfile := PwConfig.PrivateKeyFile
	keypass := PwConfig.KeyPass
	sessionpassfile := PwConfig.SessionPassFile
	passflag := "open"
	content := ""
	if len(keypass) > 0 {
		passflag = fmt.Sprintf("Encrypted:%s", keypass)
	}
	method := PwConfig.Method
	switch method {
	case typeOpenssl:
		content, err = PrivateDecryptFileSSL(cryptedfile, privatekeyfile, keypass, sessionpassfile)
	case typeGO:
		content, err = PrivateDecryptFileGo(cryptedfile, privatekeyfile, keypass)
	default:
		log.Fatalf("encryption method %s not known", method)
		os.Exit(1)
	}
	log.Debugf("Load data from %s with key %s(%s)", cryptedfile, privatekeyfile, passflag)
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
func EncryptFile() (err error) {
	cryptedFile := PwConfig.CryptedFile
	pubKeyFile := PwConfig.PubKeyFile
	plaintextfile := PwConfig.PlainTextFile
	sessionpassfile := PwConfig.SessionPassFile
	method := PwConfig.Method
	switch method {
	case typeOpenssl:
		err = PubEncryptFileSSL(plaintextfile, cryptedFile, pubKeyFile, sessionpassfile)
	case typeGO:
		err = PubEncryptFileGo(plaintextfile, cryptedFile, pubKeyFile)
	default:
		log.Fatalf("encryption method %s not known", method)
		os.Exit(1)
	}
	log.Debugf("Encrypt data from %s with key %s  into %s", plaintextfile, pubKeyFile, cryptedFile)
	if err != nil {
		log.Debug("encryption data failed")
		return
	}
	log.Debug("encrytion data success")
	return
}

// ListPasswords printout list of pwcli
func ListPasswords() (lines []string, err error) {
	log.Debugf("ListPasswords entered")
	lines, err = DecryptFile()
	if err != nil {
		log.Errorf("Decode Failed")
		return
	}
	return
}

// GetPassword ask System for data
func GetPassword(system string, account string) (password string, err error) {
	var lines []string
	log.Debugf("GetPassword for '%s'@'%s' entered", account, system)
	lines, err = DecryptFile()
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
