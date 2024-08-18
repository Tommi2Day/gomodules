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
	"github.com/ProtonMail/go-crypto/openpgp/armor"
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
func GPGUnlockKey(gpgEntity *openpgp.Entity, keypass string) (err error) {
	if gpgEntity == nil {
		err = fmt.Errorf("no key loaded")
		return
	}
	err = gpgEntity.DecryptPrivateKeys([]byte(keypass))
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
			if entityList[e].PrimaryKey == nil {
				continue
			}
			primID := entityList[e].PrimaryKey.KeyIdString()
			if primID == keyID {
				gpgEntity = entityList[e]
				log.Debugf("matched primary key Id %s", keyID)
				break
			}
			// match private subkey ID
			if entityList[e].PrivateKey == nil {
				continue
			}
			privID := entityList[e].PrivateKey.KeyIdString()
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
	err = GPGUnlockKey(entity, keypass)
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
	err = common.WriteStringToFile(targetFile, string(encryptedBytes))
	return
}

// CreateGPGEntity create GPG entity with new key pair
func CreateGPGEntity(name string, comment string, email string, passPhrase string) (entity *openpgp.Entity, privKeyID string, err error) {
	var e *openpgp.Entity

	e, err = openpgp.NewEntity(name, comment, email, nil)
	if err != nil {
		return
	}

	privKeyID = e.PrivateKey.KeyIdString()

	// need to resign self-signature with userid and add flags to make it valid
	id := ""
	for _, i := range e.Identities {
		if i.SelfSignature != nil {
			id = i.UserId.Id
			break
		}
	}
	e.Identities[id].SelfSignature.FlagSign = true
	e.Identities[id].SelfSignature.FlagCertify = true
	err = e.Identities[id].SelfSignature.SignUserId(id, e.PrimaryKey, e.PrivateKey, nil)
	if err != nil {
		err = fmt.Errorf("error selfsigning identity: %s", err)
		return
	}

	// add signing subkey
	err = e.AddSigningSubkey(nil)
	if err != nil {
		err = fmt.Errorf("error adding signing subkey: %s", err)
		return
	}

	// sign whole identity
	err = e.SignIdentity(id, e, nil)
	if err != nil {
		err = fmt.Errorf("error signing identity: %s", err)
		return
	}

	// encrypt private key
	err = e.EncryptPrivateKeys([]byte(passPhrase), nil)
	if err != nil {
		err = fmt.Errorf("error while encrypting private key: %s", err)
		return
	}
	return e, privKeyID, nil
}

// ExportGPGKeyPair export GPG entity to armored public and private key files
func ExportGPGKeyPair(entity *openpgp.Entity, publicFilename string, privFilename string) (err error) {
	var out *os.File
	var w io.WriteCloser
	if entity == nil {
		err = fmt.Errorf("no entity to export")
		return
	}
	//nolint gosec
	out, err = os.Create(publicFilename)
	w, err = armor.Encode(out, openpgp.PublicKeyType, make(map[string]string))
	if err != nil {
		err = fmt.Errorf("error creating public key file %s: %s", publicFilename, err)
		return
	}

	err = entity.Serialize(w)
	if err != nil {
		err = fmt.Errorf("error serializing public key: %s", err)
		return
	}
	_ = w.Close()

	//nolint gosec
	out, err = os.Create(privFilename)
	w, err = armor.Encode(out, openpgp.PrivateKeyType, make(map[string]string))
	if err != nil {
		err = fmt.Errorf("error creating private key file %s: %s", privFilename, err)
		return
	}
	// export withoout signg because of missing crypto.signer bug
	err = entity.SerializePrivateWithoutSigning(w, nil)
	if err != nil {
		err = fmt.Errorf("error serializing private key to %s: %s", privFilename, err)
	}
	_ = w.Close()
	return
}

/*
func createEntityFromRSAKeys(pubKey *packet.PublicKey, privKey *packet.PrivateKey,name string,comment string,email string) (entity *openpgp.Entity,err error) {
	config := packet.Config{
		DefaultHash:            crypto.SHA256,
		DefaultCipher:          packet.CipherAES256,
		DefaultCompressionAlgo: packet.NoCompression,
	}
	currentTime := config.Now()
	uid := packet.NewUserId(name, comment, email)

	e := openpgp.Entity{
		PrimaryKey: pubKey,
		PrivateKey: privKey,
		Identities: make(map[string]*openpgp.Identity),
	}
	isPrimaryId := false

	e.Identities[uid.Id] = &openpgp.Identity{
		Name:   uid.Name,
		UserId: uid,
		SelfSignature: &packet.Signature{
			CreationTime: currentTime,
			SigType:      packet.SigTypePositiveCert,
			PubKeyAlgo:   packet.PubKeyAlgoRSA,
			Hash:         config.Hash(),
			IsPrimaryId:  &isPrimaryId,
			FlagsValid:   true,
			FlagSign:     true,
			FlagCertify:  true,
			IssuerKeyId:  &e.PrimaryKey.KeyId,
		},
	}

	keyLifetimeSecs := uint32(86400 * 365)

	e.Subkeys = make([]openpgp.Subkey, 1)
	e.Subkeys[0] = openpgp.Subkey{
		PublicKey: pubKey,
		PrivateKey: privKey,
		Sig: &packet.Signature{
			CreationTime:              currentTime,
			SigType:                   packet.SigTypeSubkeyBinding,
			PubKeyAlgo:                packet.PubKeyAlgoRSA,
			Hash:                      config.Hash(),
			PreferredHash:             []uint8{8}, // SHA-256
			FlagsValid:                true,
			FlagEncryptStorage:        true,
			FlagEncryptCommunications: true,
			IssuerKeyId:               &e.PrimaryKey.KeyId,
			KeyLifetimeSecs:           &keyLifetimeSecs,
		},
	}
	return &e
}
*/
