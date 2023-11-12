package pwlib

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tommi2day/gomodules/common"

	"github.com/ProtonMail/go-crypto/openpgp"
	log "github.com/sirupsen/logrus"
)

// GPGConfig holds gpg config
type GPGConfig struct {
	StoreDir      string
	SecretKeyFile string
	SecretKeyPass string
	KeyID         string
}

// GPGUnlockKey decrypt private key and subkeys
func GPGUnlockKey(gpgEntity *openpgp.Entity, keypass string) (decryptedEntity *openpgp.Entity, err error) {
	if gpgEntity == nil {
		err = fmt.Errorf("no key loaded")
		return
	}
	// decrypt main key
	if gpgEntity.PrivateKey.Encrypted {
		if keypass == "" {
			err = fmt.Errorf("no passphrase for key %s", gpgEntity.PrivateKey.KeyIdString())
			return
		}
		err = gpgEntity.PrivateKey.Decrypt([]byte(keypass))
		if err != nil {
			err = fmt.Errorf("cannot decrypt private key for ID '%s':%s", gpgEntity.PrivateKey.KeyIdString(), err)
			return
		}
	} else {
		log.Debugf("main key %s is not encrypted", gpgEntity.PrivateKey.KeyIdString())
	}
	// decrypt sub keys
	for i, subkey := range gpgEntity.Subkeys {
		if keypass == "" {
			err = fmt.Errorf("no passphrase for key %s", subkey.PrivateKey.KeyIdString())
			return
		}
		if subkey.PrivateKey.Encrypted {
			err = subkey.PrivateKey.Decrypt([]byte(keypass))
			if err != nil {
				err = fmt.Errorf("cannot decrypt subkey %d for ID %s: %s", i, subkey.PrivateKey.KeyIdString(), err)
				return
			}
		} else {
			log.Debugf("sub key %s is not encrypted", subkey.PrivateKey.KeyIdString())
		}
	}
	decryptedEntity = gpgEntity
	return
}

// GPGSelectEntity select entity from list by Fingerprint or first one
func GPGSelectEntity(entityList openpgp.EntityList, keyID string) (gpgEntity *openpgp.Entity, err error) {
	if len(entityList) == 0 {
		err = fmt.Errorf("no gpg entity loaded")
		return
	}
	// if len(entityList) == 1 || keyID == "" {
	if keyID == "" {
		gpgEntity = entityList[0]
		log.Debugf("use first key %s", gpgEntity.PrimaryKey.KeyIdString())
	} else {
		keyID = strings.TrimPrefix(keyID, "0x")
		keyID = strings.TrimRight(keyID, "\r\n")
		for e := range entityList {
			primID := entityList[e].PrimaryKey.KeyIdString()
			privID := entityList[e].PrivateKey.KeyIdString()
			if primID == keyID {
				gpgEntity = entityList[e]
				log.Debugf("matched primary key Id %s", keyID)
				break
			}
			// match private subkey ID
			if privID == keyID {
				gpgEntity = entityList[e]
				log.Debugf("matched private key Id %s", keyID)
				break
			}
		}
		// if not found, error out
		if gpgEntity == nil {
			err = fmt.Errorf("cannot find key with id %s", keyID)
			return
		}
	}
	return
}

// GPGReadAmoredKeyRing read keyring from string
func GPGReadAmoredKeyRing(amoredKeyRing string) (entityList openpgp.EntityList, err error) {
	entityList, err = openpgp.ReadArmoredKeyRing(bytes.NewBufferString(amoredKeyRing))
	if err != nil || len(entityList) == 0 {
		if err == nil {
			err = fmt.Errorf("cannot work with entity list empty")
		} else {
			err = fmt.Errorf("cannot decode keyring string: %s", err)
		}
		return
	}
	return
}

func findGPGFiles(root string) []string {
	var a []string
	err := filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		sl := filepath.ToSlash(s)
		f := d.Name()
		ext := filepath.Ext(f)
		if ext == ".gpg" {
			a = append(a, sl)
		}
		return nil
	})
	if err != nil {
		log.Warnf("cannot walk from %s: %s", root, err)
		a = []string{}
	}
	return a
}

// GetGopassSecrets get secrets from gopass store
func GetGopassSecrets(storeRootPath string, secretKeyFile string, keypass string) (secrets string, err error) {
	var gpgid string
	var pass string

	gpgid, err = checkStoreRoot(storeRootPath)
	if err != nil {
		return
	}
	gpgFiles := findGPGFiles(storeRootPath)
	storeName := filepath.Base(storeRootPath)
	if slices.Contains([]string{".password-store", "root"}, storeName) {
		// strip base dir from name if is storePath store
		storeName = ""
	}
	for _, f := range gpgFiles {
		sn := strings.TrimSuffix(f, ".gpg")
		key := filepath.Base(sn)
		sn = strings.TrimPrefix(sn, storeRootPath+"/")
		secretPath := filepath.Dir(sn)
		secretPath = strings.ReplaceAll(secretPath, ":", "_")
		if secretPath == "." {
			secretPath = ""
		}
		if storeName != "" {
			secretPath = path.Join(storeName, secretPath)
		}
		pass, err = GPGDecryptFile(f, secretKeyFile, keypass, gpgid)
		if err == nil {
			pass = strings.TrimRight(pass, "\r\n")
			secrets += fmt.Sprintf("%s:%s:%s\n", secretPath, key, pass)
		} else {
			err = fmt.Errorf("cannot decrypt %s: %s", f, err)
			secrets = ""
			return
		}
	}
	secrets = strings.TrimRight(secrets, "\n")
	return
}

func checkStoreRoot(storeRootPath string) (gpgid string, err error) {
	if !common.IsDir(storeRootPath) {
		err = fmt.Errorf("root %s is not a directory", storeRootPath)
		return
	}
	if !common.FileExists(path.Join(storeRootPath, ".gpg-id")) {
		err = fmt.Errorf("root %s is not a gopass store", storeRootPath)
		return
	}
	gpgid, err = common.ReadFileToString(path.Join(storeRootPath, ".gpg-id"))
	return
}

// GPGDecryptFile decrypt file with GPG Key
func GPGDecryptFile(filename string, secretKeyFile string, keypass string, gpgid string) (decryptedContent string, err error) {
	var entityList openpgp.EntityList
	var entity *openpgp.Entity
	var key string
	key, err = common.ReadFileToString(secretKeyFile)
	if err != nil {
		return
	}
	entityList, err = GPGReadAmoredKeyRing(key)
	if err != nil {
		return
	}
	entity, err = GPGSelectEntity(entityList, gpgid)
	if err != nil {
		return
	}
	_, err = GPGUnlockKey(entity, keypass)
	if err != nil {
		return
	}
	encrypted := ""
	var md *openpgp.MessageDetails
	encrypted, err = common.ReadFileToString(filename)
	if err != nil {
		return
	}
	r := bytes.NewReader([]byte(encrypted))
	md, err = openpgp.ReadMessage(r, entityList, nil, nil)
	if err != nil {
		return
	}
	decryptedBytes, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return
	}
	decryptedContent = string(decryptedBytes)
	return
}

// GPGEncryptFile encrypt file with GPG Key
func GPGEncryptFile(plainFile string, targetFile string, publicKeyFile string) (err error) {
	var entityList openpgp.EntityList
	var pubKeys string
	var plain string
	var encryptedBytes []byte

	// recipients allowed to decrypt
	pubKeys, err = common.ReadFileToString(publicKeyFile)
	if err != nil {
		return
	}
	entityList, err = GPGReadAmoredKeyRing(pubKeys)
	if err != nil {
		return
	}
	plain, err = common.ReadFileToString(plainFile)
	if err != nil {
		return
	}
	encBuffer := new(bytes.Buffer)
	pw, err := openpgp.Encrypt(encBuffer, entityList, nil, &openpgp.FileHints{IsBinary: true}, nil)
	if err != nil {
		return
	}
	// write plaintext to encryptor
	_, err = pw.Write([]byte(plain))
	if err != nil {
		return
	}
	_ = pw.Close()

	// write encrypted output to file
	encryptedBytes, err = io.ReadAll(encBuffer)
	if err != nil {
		return
	}
	//nolint gosec
	err = os.WriteFile(targetFile, encryptedBytes, 0644)
	return
}
