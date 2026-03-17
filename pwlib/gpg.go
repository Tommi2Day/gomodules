package pwlib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tommi2day/gomodules/common"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
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

// GPGEncryptFileMulti encrypts plainFile to targetFile for all GPG entities loaded
// from pubKeyFiles. Each file must contain one or more armored public keys.
// The resulting file can be decrypted by any of the corresponding private keys.
func GPGEncryptFileMulti(plainFile, targetFile string, pubKeyFiles []string) error {
	if len(pubKeyFiles) == 0 {
		return fmt.Errorf("no public key files provided")
	}
	var entityList openpgp.EntityList
	for _, kf := range pubKeyFiles {
		pubKeys, err := common.ReadFileToString(kf)
		if err != nil {
			return fmt.Errorf("failed to read public key file '%s': %v", kf, err)
		}
		el, err := GPGReadAmoredKeyRing(pubKeys)
		if err != nil {
			return fmt.Errorf("failed to parse public key file '%s': %v", kf, err)
		}
		entityList = append(entityList, el...)
	}
	plain, err := common.ReadFileToString(plainFile)
	if err != nil {
		return fmt.Errorf("failed to read plain file: %v", err)
	}
	encBuffer := new(bytes.Buffer)
	pw, err := openpgp.Encrypt(encBuffer, entityList, nil, &openpgp.FileHints{IsBinary: true}, nil)
	if err != nil {
		return fmt.Errorf("failed to initialise GPG encryptor: %v", err)
	}
	if _, err = pw.Write([]byte(plain)); err != nil {
		return fmt.Errorf("failed to write encrypted content: %v", err)
	}
	_ = pw.Close()
	encryptedBytes, err := io.ReadAll(encBuffer)
	if err != nil {
		return fmt.Errorf("failed to read encrypted buffer: %v", err)
	}
	if err = common.WriteStringToFile(targetFile, string(encryptedBytes)); err != nil {
		return fmt.Errorf("failed to write encrypted file '%s': %v", targetFile, err)
	}
	log.Debugf("GPG-encrypted %s → %s for %d key file(s)", plainFile, targetFile, len(pubKeyFiles))
	return nil
}

// GPGDecryptFileMulti decrypts filename using the first matching private key found in
// secretKeyFiles. It delegates detection to GPGFindDecryptKey and decryption to
// GPGDecryptFile. Returns an error if no key matches.
func GPGDecryptFileMulti(filename string, secretKeyFiles []string, keypass string) (string, error) {
	matched, err := GPGFindDecryptKey(filename, secretKeyFiles)
	if err != nil {
		return "", err
	}
	return GPGDecryptFile(filename, matched, keypass, "")
}

// GPGSignFile signs a file using a private GPG key
func GPGSignFile(plainFile string, signatureFile string, secretKeyFile string, keypass string) error {
	log.Debugf("Sign %s with GPG private key %s", plainFile, secretKeyFile)
	key, err := common.ReadFileToString(secretKeyFile)
	if err != nil {
		return err
	}
	entityList, err := GPGReadAmoredKeyRing(key)
	if err != nil {
		return err
	}
	entity, err := GPGSelectEntity(entityList, "")
	if err != nil {
		return err
	}
	err = GPGUnlockKey(entity, keypass)
	if err != nil {
		return err
	}

	plain, err := os.Open(plainFile)
	if err != nil {
		return err
	}
	defer func(plain *os.File) {
		_ = plain.Close()
	}(plain)

	sigFile, err := os.Create(signatureFile)
	if err != nil {
		return err
	}
	defer func(sigFile *os.File) {
		_ = sigFile.Close()
	}(sigFile)

	err = openpgp.ArmoredDetachSign(sigFile, entity, plain, nil)
	if err != nil {
		return fmt.Errorf("failed to sign: %v", err)
	}

	return nil
}

// GPGVerifyFile verifies a GPG signature
func GPGVerifyFile(plainFile string, signatureFile string, publicKeyFile string) (bool, error) {
	log.Debugf("Verify %s with GPG public key %s", plainFile, publicKeyFile)
	key, err := common.ReadFileToString(publicKeyFile)
	if err != nil {
		return false, err
	}
	entityList, err := GPGReadAmoredKeyRing(key)
	if err != nil {
		return false, err
	}

	plain, err := os.Open(plainFile)
	if err != nil {
		return false, err
	}
	defer func(plain *os.File) {
		_ = plain.Close()
	}(plain)

	sig, err := os.Open(signatureFile)
	if err != nil {
		return false, err
	}
	defer func(sig *os.File) {
		_ = sig.Close()
	}(sig)

	signer, err := openpgp.CheckArmoredDetachedSignature(entityList, plain, sig, nil)
	if err != nil {
		return false, fmt.Errorf("signature verification failed: %v", err)
	}
	if signer == nil {
		return false, fmt.Errorf("no signer found")
	}

	return true, nil
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
	_ = out.Close()

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
	_ = out.Close()
	return
}

// GPGDetectRecipients reads the OpenPGP packet header of an encrypted file and
// returns the 64-bit key IDs of all explicit recipients (PKESK packets).
// Files encrypted with wildcard (hidden) recipients return an empty slice and an error.
func GPGDetectRecipients(filename string) (keyIDs []string, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("read encrypted file %s failed: %w", filename, err)
		return
	}
	pktReader := packet.NewReader(bytes.NewReader(data))
	seen := make(map[uint64]bool)
	for {
		p, pErr := pktReader.Next()
		if pErr == io.EOF {
			break
		}
		if pErr != nil {
			// stop at the first parse error – this is usually the encrypted
			// data packet itself, which we cannot read without the session key
			break
		}
		ek, ok := p.(*packet.EncryptedKey)
		if !ok {
			continue
		}
		if ek.KeyId != 0 && !seen[ek.KeyId] {
			keyIDs = append(keyIDs, fmt.Sprintf("%016X", ek.KeyId))
			seen[ek.KeyId] = true
		}
	}
	if len(keyIDs) == 0 {
		err = fmt.Errorf("no explicit recipient key IDs found in %s (file may use wildcard/hidden recipients)", filename)
	}
	log.Debugf("detected %d recipient key ID(s) in %s", len(keyIDs), filename)
	return
}

// GPGFindDecryptKey searches keyFiles for an armored key whose primary key or
// encryption subkey matches a recipient ID embedded in the encrypted file at
// filename. It returns the path of the first matching key file.
func GPGFindDecryptKey(filename string, keyFiles []string) (matchedFile string, err error) {
	keyIDs, err := GPGDetectRecipients(filename)
	if err != nil {
		return
	}
	for _, keyFile := range keyFiles {
		content, rErr := common.ReadFileToString(keyFile)
		if rErr != nil {
			log.Warnf("GPGFindDecryptKey: skip unreadable key file %s: %v", keyFile, rErr)
			continue
		}
		entityList, rErr := GPGReadAmoredKeyRing(content)
		if rErr != nil {
			log.Warnf("GPGFindDecryptKey: skip invalid key file %s: %v", keyFile, rErr)
			continue
		}
		if gpgEntityMatchesKeyIDs(entityList, keyIDs) {
			matchedFile = keyFile
			log.Debugf("GPGFindDecryptKey: matched key file %s for %s", keyFile, filename)
			return
		}
	}
	err = fmt.Errorf("no matching GPG key found among %d key file(s) for %s", len(keyFiles), filename)
	return
}

// gpgEntityMatchesKeyIDs reports whether any entity in the list has a primary
// key or subkey whose ID appears in keyIDs.
func gpgEntityMatchesKeyIDs(entityList openpgp.EntityList, keyIDs []string) bool {
	for _, e := range entityList {
		if gpgKeyIDInSlice(fmt.Sprintf("%016X", e.PrimaryKey.KeyId), keyIDs) {
			return true
		}
		for _, sk := range e.Subkeys {
			if gpgKeyIDInSlice(fmt.Sprintf("%016X", sk.PublicKey.KeyId), keyIDs) {
				return true
			}
		}
	}
	return false
}

func gpgKeyIDInSlice(keyID string, keyIDs []string) bool {
	for _, k := range keyIDs {
		if k == keyID {
			return true
		}
	}
	return false
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
