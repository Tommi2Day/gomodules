package pwlib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tommi2day/gomodules/common"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	log "github.com/sirupsen/logrus"
)

const (
	gpgEnvHome           = "GNUPGHOME"   //nolint:gosec // env-var name, not a credential
	gpgDefaultHomeDir    = ".gnupg"      //nolint:gosec // path constant, not a credential
	gpgSecretKeyRingFile = "secring.gpg" //nolint:gosec // path constant, not a credential
)

// GPGHomeDir returns the GnuPG home directory.
// GNUPGHOME overrides the default ~/.gnupg.
func GPGHomeDir() (string, error) {
	if h := os.Getenv(gpgEnvHome); h != "" {
		log.Debugf("GPGHomeDir: using %s=%s", gpgEnvHome, h)
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, gpgDefaultHomeDir), nil
}

// GPGSecretKeyRingPath returns the path to the user's GPG secret keyring (secring.gpg).
func GPGSecretKeyRingPath() (string, error) {
	gpgHome, err := GPGHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(gpgHome, gpgSecretKeyRingFile), nil
}

// GPGReadSecretKeyRing reads a binary GPG secret keyring file (secring.gpg format).
// The returned entities may still have encrypted private keys; unlock with DecryptPrivateKeys.
func GPGReadSecretKeyRing(keyRingPath string) (openpgp.EntityList, error) {
	f, err := os.Open(keyRingPath) //nolint:gosec // path comes from caller / env
	if err != nil {
		return nil, fmt.Errorf("open GPG keyring %s: %w", keyRingPath, err)
	}
	defer func() { _ = f.Close() }()
	el, err := openpgp.ReadKeyRing(f)
	if err != nil {
		return nil, fmt.Errorf("read GPG keyring %s: %w", keyRingPath, err)
	}
	if len(el) == 0 {
		return nil, fmt.Errorf("no keys found in GPG keyring %s", keyRingPath)
	}
	log.Debugf("read %d GPG key(s) from keyring %s", len(el), keyRingPath)
	return el, nil
}

// GPGExportSecretKeysArmored exports all secret keys from the system GPG keyring
// by calling the gpg binary with --export-secret-keys --armor.
// This is the reliable path for modern GnuPG (2.1+) that uses private-keys-v1.d.
func GPGExportSecretKeysArmored() (string, error) {
	out, err := exec.Command("gpg", "--export-secret-keys", "--armor").Output() //nolint:gosec // fixed args, no user input
	if err != nil {
		return "", fmt.Errorf("gpg --export-secret-keys failed: %w", err)
	}
	if len(out) == 0 {
		return "", fmt.Errorf("no GPG secret keys exported (keyring may be empty)")
	}
	return string(out), nil
}

// GPGSystemSecretKeys returns all secret keys available in the user's GPG keyring.
// It first tries to read secring.gpg (legacy / gopass-managed), then falls back
// to calling the gpg binary (modern GnuPG 2.1+ using private-keys-v1.d).
// The returned entities may have encrypted private keys.
func GPGSystemSecretKeys() (openpgp.EntityList, error) {
	keyRingPath, pathErr := GPGSecretKeyRingPath()
	if pathErr == nil {
		if el, err := GPGReadSecretKeyRing(keyRingPath); err == nil {
			return el, nil
		}
	}
	// fall back to gpg binary export
	armored, err := GPGExportSecretKeysArmored()
	if err != nil {
		return nil, fmt.Errorf("cannot load GPG secret keys from keyring or gpg binary: %w", err)
	}
	el, err := GPGReadAmoredKeyRing(armored)
	if err != nil {
		return nil, fmt.Errorf("parse GPG keys from binary export: %w", err)
	}
	log.Debugf("loaded %d GPG key(s) via gpg binary export", len(el))
	return el, nil
}

// GPGDecryptFileAuto decrypts filename using all secret keys found in the user's
// GPG keyring (secring.gpg or gpg binary export). passphrase is used to unlock
// any encrypted private keys. When passphrase is empty the GPG_PASSPHRASE
// environment variable is tried; if the key is still locked GPGAgentDecrypt is
// called to use gpg-agent via the native Go Assuan client.
func GPGDecryptFileAuto(filename, passphrase string) (string, error) {
	entityList, err := GPGSystemSecretKeys()
	if err != nil {
		return "", err
	}
	for _, e := range entityList {
		_ = e.DecryptPrivateKeys([]byte(passphrase))
	}
	if passphrase == "" && gpgAnyKeyEncrypted(entityList) {
		passphrase = os.Getenv("GPG_PASSPHRASE") //nolint:gosec // env-var name, not a credential
		if passphrase != "" {
			log.Debugf("GPGDecryptFileAuto: using GPG_PASSPHRASE env var to unlock key(s)")
			for _, e := range entityList {
				_ = e.DecryptPrivateKeys([]byte(passphrase))
			}
		}
	}
	// If keys are still encrypted, delegate to gpg-agent via the Assuan protocol.
	if gpgAnyKeyEncrypted(entityList) {
		log.Debugf("GPGDecryptFileAuto: key(s) still encrypted, delegating to gpg-agent")
		return GPGAgentDecrypt(filename, entityList)
	}
	encrypted, err := os.ReadFile(filename) //nolint:gosec // path comes from caller
	if err != nil {
		return "", fmt.Errorf("read encrypted file %s: %w", filename, err)
	}
	md, err := openpgp.ReadMessage(bytes.NewReader(encrypted), entityList, nil, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt %s with system GPG keys: %w", filename, err)
	}
	decrypted, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}
	log.Debugf("auto-decrypted %s with system GPG key", filename)
	return string(decrypted), nil
}

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
	if keypass == "" && gpgAnyKeyEncrypted(openpgp.EntityList{entity}) {
		if envPass := os.Getenv("GPG_PASSPHRASE"); envPass != "" { //nolint:gosec // env-var name, not a credential
			keypass = envPass
			log.Debugf("GPGDecryptFile: using GPG_PASSPHRASE env var to unlock key %s", entity.PrimaryKey.KeyIdString())
		}
	}
	if keypass != "" {
		err = GPGUnlockKey(entity, keypass)
		if err != nil {
			return
		}
	}
	// If the key is still encrypted (no passphrase supplied or unlock failed), try gpg-agent.
	if gpgAnyKeyEncrypted(openpgp.EntityList{entity}) {
		log.Debugf("GPGDecryptFile: key %s still encrypted, delegating to gpg-agent", entity.PrimaryKey.KeyIdString())
		decryptedContent, err = GPGAgentDecrypt(filename, openpgp.EntityList{entity})
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
		_ = pw.Close()
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
		_ = pw.Close()
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
		_ = out.Close()
		err = fmt.Errorf("error creating public key file %s: %s", publicFilename, err)
		return
	}

	err = entity.Serialize(w)
	if err != nil {
		_ = w.Close()
		_ = out.Close()
		err = fmt.Errorf("error serializing public key: %s", err)
		return
	}
	_ = w.Close()
	_ = out.Close()

	//nolint gosec
	out, err = os.Create(privFilename)
	w, err = armor.Encode(out, openpgp.PrivateKeyType, make(map[string]string))
	if err != nil {
		_ = out.Close()
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

// gpgAnyKeyEncrypted reports whether any entity in the list has an encrypted
// primary private key or an encrypted subkey.
func gpgAnyKeyEncrypted(entityList openpgp.EntityList) bool {
	for _, e := range entityList {
		if e.PrivateKey != nil && e.PrivateKey.Encrypted {
			return true
		}
		for _, sk := range e.Subkeys {
			if sk.PrivateKey != nil && sk.PrivateKey.Encrypted {
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
