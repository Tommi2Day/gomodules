package pwlib

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"filippo.io/age"
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
