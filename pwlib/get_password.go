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
var Methods = []string{typeGO, typeOpenssl, typeEnc, typePlain, typeVault, typeGPG, typeGopass, typeKMS}

// DecryptFile decripts an rsa protected file
func (pc *PassConfig) DecryptFile() (lines []string, err error) {
	cryptedfile := pc.CryptedFile
	privatekeyfile := pc.PrivateKeyFile
	keypass := pc.KeyPass
	datadir := pc.DataDir
	sessionpassfile := pc.SessionPassFile
	passflag := "open"
	content := ""
	method := pc.Method
	keyID := pc.KMSKeyID
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
		content, err = common.ReadFileToString(cryptedfile)
	case typeVault:
		content, err = GetVaultSecret(cryptedfile, "", "")
	case typeGPG:
		content, err = GPGDecryptFile(cryptedfile, privatekeyfile, keypass, "")
	case typeGopass:
		content, err = GetGopassSecrets(datadir, privatekeyfile, keypass)
	case typeKMS:
		content, err = KMSDecryptFile(cryptedfile, keyID, sessionpassfile)
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
	keyID := pc.KMSKeyID
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
	case typeGPG:
		err = GPGEncryptFile(plaintextfile, cryptedFile, pubKeyFile)
	case typeKMS:
		err = KMSEncryptFile(plaintextfile, cryptedFile, keyID, sessionpassfile)
	case typeVault, typeGopass:
		// not implemented yet
		err = fmt.Errorf("encryption method %s not implemented yet", method)
	default:
		log.Fatalf("Enc method %s not known", method)
		os.Exit(1)
	}

	if err != nil {
		log.Debug("encryption data failed")
		return
	}
	log.Debug("encryption data success")
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
	if pc.Method == typeVault {
		// in vault mode we use cryptedfile to handover vault path
		pc.CryptedFile = system
	}
	lines, err = pc.DecryptFile()
	if err != nil {
		return
	}
	found := false
	direct := false
	if pc.Method == typeVault {
		// in vault mode we need to replace ":" in system = vault path to match
		system = strings.ReplaceAll(system, ":", "_")
		pc.CaseSensitive = true
	}

	if pc.Method == typeGopass {
		pc.CaseSensitive = true
	}
	// match strings in function to make linter happy
	password, found, direct = pc.match(lines, system, account)
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

func (pc *PassConfig) match(lines []string, system string, account string) (password string, found bool, direct bool) {
	password = ""
	found = false
	direct = false

	for _, line := range lines {
		if common.CheckSkip(line) {
			continue
		}

		fields := strings.SplitN(line, ":", 3)
		if len(fields) != 3 {
			log.Debugf("Skip incomplete record %s", line)
			continue
		}

		if pc.isDirectMatch(fields, system, account) {
			log.Debug("Found direct match")
			password, found, direct = fields[2], true, true
			break
		}

		if pc.isDefaultMatch(fields, account) {
			log.Debug("Found new default match candidate")
			password, found = fields[2], true
		}
	}
	return
}

func (pc *PassConfig) isDirectMatch(fields []string, system string, account string) bool {
	if pc.CaseSensitive {
		return system == fields[0] && account == fields[1]
	}
	return strings.EqualFold(system, fields[0]) && strings.EqualFold(account, fields[1])
}

func (pc *PassConfig) isDefaultMatch(fields []string, account string) bool {
	const defaultSystem = "!default"
	if pc.Method == typeVault || pc.Method == typeGopass {
		return false
	}
	if pc.CaseSensitive {
		return strings.ToLower(fields[0]) == defaultSystem && account == fields[1]
	}
	return strings.EqualFold(fields[0], defaultSystem) && strings.EqualFold(account, fields[1])
}
