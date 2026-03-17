package pwlib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	log "github.com/sirupsen/logrus"
	"github.com/tommi2day/gomodules/common"
)

// AgeConfig holds age config
type AgeConfig struct {
	StoreDir      string
	SecretKeyFile string
	SecretKeyPass string
	PublicKey     string
}

// CreateAgeIdentity creates a new age key pair
func CreateAgeIdentity() (identity *age.X25519Identity, recipient string, err error) {
	identity, err = age.GenerateX25519Identity()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate age identity: %v", err)
	}
	recipient = identity.Recipient().String()
	return
}

// ExportAgeKeyPair exports age identity to files
func ExportAgeKeyPair(identity *age.X25519Identity, publicFilename string, privFilename string) error {
	if identity == nil {
		return fmt.Errorf("no identity to export")
	}

	// Export private key
	err := common.WriteStringToFile(privFilename, identity.String())
	if err != nil {
		return fmt.Errorf("error writing private key to %s: %v", privFilename, err)
	}

	// Export public key (recipient)
	err = common.WriteStringToFile(publicFilename, identity.Recipient().String())
	if err != nil {
		return fmt.Errorf("error writing public key to %s: %v", publicFilename, err)
	}

	return nil
}

/*
func cleanAgeKeys(content string) []string {
	// Clean up the identity content by removing comments and empty lines
	var cleanedContent []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cleanedContent = append(cleanedContent, line)
	}
	return cleanedContent
}
*/
// AgeDecryptFile decrypts a file using an age identity
func AgeDecryptFile(filename string, identityFile string) (decryptedContent string, err error) {
	decryptedContent = ""
	// Read private key
	keyFile, err := os.Open(identityFile)
	if err != nil {
		err = fmt.Errorf("failed to open identity file '%s: %v", identityFile, err)
		return
	}
	identities, err := age.ParseIdentities(keyFile)
	if err != nil {
		err = fmt.Errorf("failed to parse identity file '%s': %v", identityFile, err)
		return
	}

	// Read encrypted file
	encrypted, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read encrypted file: %v", err)
	}

	// Decrypt
	r, err := age.Decrypt(bytes.NewReader(encrypted), identities...)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt using identities from '%s': %v", identityFile, err)
	}

	// Read decrypted content
	decryptedBytes, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted content: %v", err)
	}

	return string(decryptedBytes), nil
}

// AgeDetectIdentity searches identityFiles for the age identity that can unwrap
// the file key in the age-encrypted file at filename.
// It returns the path of the first matching identity file.
// The matching is done by attempting to unwrap the file key from the age header;
// the encrypted payload is never read.
func AgeDetectIdentity(filename string, identityFiles []string) (matchedFile string, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("read encrypted file %s failed: %w", filename, err)
		return
	}
	for _, idFile := range identityFiles {
		f, fErr := os.Open(idFile)
		if fErr != nil {
			log.Debugf("AgeDetectIdentity: skip unreadable identity file %s: %v", idFile, fErr)
			continue
		}
		ids, pErr := age.ParseIdentities(f)
		_ = f.Close()
		if pErr != nil {
			log.Debugf("AgeDetectIdentity: skip invalid identity file %s: %v", idFile, pErr)
			continue
		}
		// age.Decrypt resolves the file key from the header stanzas synchronously.
		// If the identity cannot unwrap the file key it returns an error before
		// any payload bytes are read, making this an efficient header-only check.
		if _, dErr := age.Decrypt(bytes.NewReader(data), ids...); dErr == nil {
			matchedFile = idFile
			log.Debugf("AgeDetectIdentity: matched identity file %s for %s", idFile, filename)
			return
		}
	}
	err = fmt.Errorf("no matching age identity found among %d identity file(s) for %s", len(identityFiles), filename)
	return
}

// ExportAgeKeyPairEncrypted exports an age identity to files, with the private
// key encrypted by a passphrase using the age scrypt (passphrase) recipient.
// The public key file is written in plaintext; the private key file is an
// age-encrypted binary whose content is the AGE-SECRET-KEY-1 string.
func ExportAgeKeyPairEncrypted(identity *age.X25519Identity, pubFile, encPrivFile, passphrase string) error {
	if identity == nil {
		return fmt.Errorf("no identity to export")
	}
	if err := common.WriteStringToFile(pubFile, identity.Recipient().String()); err != nil {
		return fmt.Errorf("error writing public key to %s: %v", pubFile, err)
	}
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return fmt.Errorf("failed to create scrypt recipient: %v", err)
	}
	//nolint gosec
	f, err := os.Create(encPrivFile)
	if err != nil {
		return fmt.Errorf("failed to create encrypted identity file %s: %v", encPrivFile, err)
	}
	defer func() { _ = f.Close() }()
	w, err := age.Encrypt(f, recipient)
	if err != nil {
		return fmt.Errorf("failed to initialise identity encryptor: %v", err)
	}
	if _, err = w.Write([]byte(identity.String())); err != nil {
		return fmt.Errorf("failed to write encrypted identity: %v", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to finalise identity encryption: %v", err)
	}
	log.Debugf("exported encrypted age identity to %s", encPrivFile)
	return nil
}

// AgeLoadEncryptedIdentity decrypts a passphrase-protected age identity file
// and returns the contained X25519Identity.
// The identity file must have been created with ExportAgeKeyPairEncrypted.
func AgeLoadEncryptedIdentity(encIdentityFile, passphrase string) (identity *age.X25519Identity, err error) {
	data, err := os.ReadFile(encIdentityFile)
	if err != nil {
		err = fmt.Errorf("failed to read encrypted identity file %s: %v", encIdentityFile, err)
		return
	}
	scryptID, sErr := age.NewScryptIdentity(passphrase)
	if sErr != nil {
		err = fmt.Errorf("failed to create scrypt identity: %v", sErr)
		return
	}
	r, dErr := age.Decrypt(bytes.NewReader(data), scryptID)
	if dErr != nil {
		err = fmt.Errorf("failed to decrypt identity file %s (wrong passphrase?): %v", encIdentityFile, dErr)
		return
	}
	keyBytes, rErr := io.ReadAll(r)
	if rErr != nil {
		err = fmt.Errorf("failed to read decrypted identity: %v", rErr)
		return
	}
	identity, err = age.ParseX25519Identity(strings.TrimSpace(string(keyBytes)))
	if err != nil {
		err = fmt.Errorf("failed to parse decrypted identity from %s: %v", encIdentityFile, err)
	}
	log.Debugf("loaded encrypted age identity from %s", encIdentityFile)
	return
}

// AgeDecryptFileWithEncryptedIdentity decrypts filename using an age identity
// file that is itself passphrase-protected (created with ExportAgeKeyPairEncrypted).
func AgeDecryptFileWithEncryptedIdentity(filename, encIdentityFile, passphrase string) (decryptedContent string, err error) {
	identity, err := AgeLoadEncryptedIdentity(encIdentityFile, passphrase)
	if err != nil {
		return
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("failed to read encrypted file %s: %v", filename, err)
		return
	}
	r, dErr := age.Decrypt(bytes.NewReader(data), identity)
	if dErr != nil {
		err = fmt.Errorf("failed to decrypt %s with encrypted identity: %v", filename, dErr)
		return
	}
	decryptedBytes, rErr := io.ReadAll(r)
	if rErr != nil {
		err = fmt.Errorf("failed to read decrypted content: %v", rErr)
		return
	}
	decryptedContent = string(decryptedBytes)
	log.Debugf("decrypted %s using encrypted identity %s", filename, encIdentityFile)
	return
}

// AgeEncryptFileWithPassphrase encrypts a file using a passphrase (age scrypt
// recipient). The resulting file can only be decrypted with the same passphrase;
// it cannot be combined with public-key recipients.
func AgeEncryptFileWithPassphrase(plainFile, targetFile, passphrase string) error {
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return fmt.Errorf("failed to create scrypt recipient: %v", err)
	}
	plain, err := common.ReadFileToString(plainFile)
	if err != nil {
		return fmt.Errorf("failed to read plain file: %v", err)
	}
	//nolint gosec
	f, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create encrypted file '%s': %v", targetFile, err)
	}
	w, err := age.Encrypt(f, recipient)
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to initialise encryptor: %v", err)
	}
	if _, err = w.Write([]byte(plain)); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write encrypted content: %v", err)
	}
	if err = w.Close(); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to finalise encryption: %v", err)
	}
	_ = f.Close()
	log.Debugf("passphrase-encrypted %s → %s", plainFile, targetFile)
	return nil
}

// AgeDecryptFileWithPassphrase decrypts an age file that was encrypted with a
// passphrase (age scrypt recipient) using AgeEncryptFileWithPassphrase.
func AgeDecryptFileWithPassphrase(filename, passphrase string) (decryptedContent string, err error) {
	identity, iErr := age.NewScryptIdentity(passphrase)
	if iErr != nil {
		err = fmt.Errorf("failed to create scrypt identity: %v", iErr)
		return
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("failed to read encrypted file %s: %v", filename, err)
		return
	}
	r, dErr := age.Decrypt(bytes.NewReader(data), identity)
	if dErr != nil {
		err = fmt.Errorf("failed to decrypt %s with passphrase: %v", filename, dErr)
		return
	}
	decryptedBytes, rErr := io.ReadAll(r)
	if rErr != nil {
		err = fmt.Errorf("failed to read decrypted content: %v", rErr)
		return
	}
	decryptedContent = string(decryptedBytes)
	log.Debugf("passphrase-decrypted %s", filename)
	return
}

// AgeDetectIdentityWithPassphrase searches encryptedIdentityFiles for a
// passphrase-protected age identity (created with ExportAgeKeyPairEncrypted)
// that can decrypt filename. All files are unlocked with the same passphrase.
func AgeDetectIdentityWithPassphrase(filename string, encryptedIdentityFiles []string, passphrase string) (matchedFile string, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("read encrypted file %s failed: %w", filename, err)
		return
	}
	for _, idFile := range encryptedIdentityFiles {
		identity, loadErr := AgeLoadEncryptedIdentity(idFile, passphrase)
		if loadErr != nil {
			log.Debugf("AgeDetectIdentityWithPassphrase: skip %s: %v", idFile, loadErr)
			continue
		}
		if _, dErr := age.Decrypt(bytes.NewReader(data), identity); dErr == nil {
			matchedFile = idFile
			log.Debugf("AgeDetectIdentityWithPassphrase: matched %s for %s", idFile, filename)
			return
		}
	}
	err = fmt.Errorf("no matching encrypted age identity found among %d file(s) for %s", len(encryptedIdentityFiles), filename)
	return
}

// AgeDecryptFileAuto decrypts filename using an age identity file that may be
// either a plaintext identity (AGE-SECRET-KEY-1…) or a passphrase-protected
// identity file created with ExportAgeKeyPairEncrypted.
//
// Detection is based on whether the identity file can be parsed as a plaintext
// age identity:
//   - plaintext identity → calls AgeDecryptFile (passphrase is ignored)
//   - binary/encrypted identity → calls AgeDecryptFileWithEncryptedIdentity
//     using passphrase; returns an error if passphrase is empty
func AgeDecryptFileAuto(filename, identityFile, passphrase string) (string, error) {
	f, err := os.Open(identityFile)
	if err != nil {
		return "", fmt.Errorf("cannot open identity file %s: %v", identityFile, err)
	}
	_, parseErr := age.ParseIdentities(f)
	_ = f.Close()

	if parseErr == nil {
		// plaintext identity file
		log.Debugf("AgeDecryptFileAuto: %s is a plaintext identity", identityFile)
		return AgeDecryptFile(filename, identityFile)
	}
	// identity file is not valid plaintext — assume it is passphrase-protected
	if passphrase == "" {
		return "", fmt.Errorf("identity file %s is not a valid plaintext identity and no passphrase was provided: %v", identityFile, parseErr)
	}
	log.Debugf("AgeDecryptFileAuto: %s appears to be a passphrase-protected identity", identityFile)
	return AgeDecryptFileWithEncryptedIdentity(filename, identityFile, passphrase)
}

// AgeEncryptFile encrypts a file using an age recipient
func AgeEncryptFile(plainFile string, targetFile string, recipientsFile string) error {
	// Read recipient (public key)
	recFile, err := os.Open(recipientsFile)
	if err != nil {
		err = fmt.Errorf("failed to open recipients file '%s: %v", recipientsFile, err)
		return err
	}
	recipients, err := age.ParseRecipients(recFile)
	_ = recFile.Close()
	if err != nil {
		return fmt.Errorf("failed to parse recipient file '%s': %v", recipientsFile, err)
	}

	// Read plain content
	plain, err := common.ReadFileToString(plainFile)
	if err != nil {
		return fmt.Errorf("failed to read plain file: %v", err)
	}

	// Create encrypted file
	encryptedFile, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create encrypted file '%s': %v", targetFile, err)
	}

	// Create encryptor
	w, err := age.Encrypt(encryptedFile, recipients...)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %v", err)
	}

	// Write and encrypt content
	if _, err = w.Write([]byte(plain)); err != nil {
		return fmt.Errorf("failed to write encrypted content: %v", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to finalize encryption: %v", err)
	}
	_ = encryptedFile.Close()
	return nil
}

// AgeEncryptFileMulti encrypts plainFile to targetFile for all recipients collected
// from recipientFiles. Each file may contain one or more age public keys (one per line).
// The resulting file can be decrypted by any of the corresponding private keys.
func AgeEncryptFileMulti(plainFile, targetFile string, recipientFiles []string) error {
	if len(recipientFiles) == 0 {
		return fmt.Errorf("no recipient files provided")
	}
	var recipients []age.Recipient
	for _, rf := range recipientFiles {
		f, err := os.Open(rf)
		if err != nil {
			return fmt.Errorf("failed to open recipients file '%s': %v", rf, err)
		}
		recs, err := age.ParseRecipients(f)
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("failed to parse recipients file '%s': %v", rf, err)
		}
		recipients = append(recipients, recs...)
	}
	plain, err := common.ReadFileToString(plainFile)
	if err != nil {
		return fmt.Errorf("failed to read plain file: %v", err)
	}
	//nolint:gosec
	f, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create encrypted file '%s': %v", targetFile, err)
	}
	w, err := age.Encrypt(f, recipients...)
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to initialise encryptor: %v", err)
	}
	if _, err = w.Write([]byte(plain)); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write encrypted content: %v", err)
	}
	if err = w.Close(); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to finalise encryption: %v", err)
	}
	_ = f.Close()
	log.Debugf("age-encrypted %s → %s for %d recipient file(s)", plainFile, targetFile, len(recipientFiles))
	return nil
}

// AgeDecryptFileMulti decrypts filename using the first matching identity found in
// identityFiles. It delegates detection to AgeDetectIdentity and decryption to
// AgeDecryptFile. Returns an error if no identity matches.
func AgeDecryptFileMulti(filename string, identityFiles []string) (string, error) {
	matched, err := AgeDetectIdentity(filename, identityFiles)
	if err != nil {
		return "", err
	}
	return AgeDecryptFile(filename, matched)
}
