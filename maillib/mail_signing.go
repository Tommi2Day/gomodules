// Package maillib provides functions for mail operations including signing and verification.
//
// # Mail Signing and Verification
//
// This package supports multiple cryptographic methods for signing and verifying mail content:
//
// RSA (SigningMethodRSA):
// - Uses RSA-2048 key pairs with SHA-256 hashing
// - Signatures are Base64-encoded
// - Fast and widely supported method
// - Best for: General-purpose email signing
//
// ECDSA (SigningMethodECDSA):
// - Uses Elliptic Curve keys (preferably P-256 or P-384)
// - Smaller signatures than RSA
// - Good performance characteristics
// - Best for: Performance-critical applications
//
// GPG (SigningMethodGPG):
// - Uses GnuPG keys (compatible with OpenPGP standard)
// - Requires GPG installation or key files
// - Widely supported in traditional email systems
// - Best for: Email systems that need OpenPGP compliance
//
// S/MIME (SigningMethodSMIME):
// - Uses X.509 certificates
// - Enterprise-standard email signing method
// - Supports certificate chains
// - Best for: Corporate email systems
//
// Usage Example:
//
//	// Create a mail
//	mail := NewMail("sender@example.com", "recipient@example.com")
//	mail.TextParts = []string{"Hello, this is a signed message"}
//
//	// Configure signing with RSA
//	config := &MailSignatureConfig{
//	    Method:         SigningMethodRSA,
//	    PrivateKeyFile: "/path/to/private.key",
//	    PublicKeyFile:  "/path/to/public.key",
//	    KeyPassphrase:  "password123",
//	}
//
//	// Sign the mail
//	err := mail.SignMail(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Verify the signature (e.g., when receiving)
//	valid, err := mail.VerifyMailSignature()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if valid {
//	    fmt.Println("Signature verified successfully!")
//	}
package maillib

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/pwlib"
)

// SigningMethod defines the mail signing method
type SigningMethod string

const (
	SigningMethodRSA   SigningMethod = "rsa"
	SigningMethodECDSA SigningMethod = "ecdsa"
	SigningMethodGPG   SigningMethod = "gpg"
	SigningMethodSMIME SigningMethod = "smime"
	pemTypeCertificate               = "CERTIFICATE"
)

// MailSignatureConfig holds signing configuration
type MailSignatureConfig struct {
	Method           SigningMethod
	PrivateKeyFile   string   // Private key or Certificate file for S/MIME
	PublicKeyFile    string   // Public key or Certificate file for verification
	KeyPassphrase    string   // Passphrase for encrypted private key
	SignMessageBody  bool     // For S/MIME: sign only body or entire message
	CertificateChain []string // For S/MIME: optional Certificate Chain
	IncludeChain     bool     // Include cert chain in S/MIME signature
}

// SignMailContent signs the mail content with the selected method
func SignMailContent(content string, config *MailSignatureConfig) (signature string, err error) {
	if config == nil {
		return "", fmt.Errorf("signature config is nil")
	}
	log.Debugf("Signing mail content with method: %s", config.Method)

	switch config.Method {
	case SigningMethodRSA:
		signature, err = pwlib.RsaSignString(content, config.PrivateKeyFile, config.KeyPassphrase)
	case SigningMethodECDSA:
		signature, err = pwlib.EcdsaSignString(content, config.PrivateKeyFile, config.KeyPassphrase)
	case SigningMethodGPG:
		signature, err = signContentWithGPG(content, config)
	case SigningMethodSMIME:
		signature, err = signContentWithSMIME(content, config)
	default:
		err = fmt.Errorf("unsupported signing method: %s", config.Method)
	}

	if err != nil {
		log.Debugf("Signing failed: %v", err)
	}
	return
}

// VerifyMailSignature verifies a mail signature
func VerifyMailSignature(content string, signature string, config *MailSignatureConfig) (valid bool, err error) {
	if config == nil {
		return false, fmt.Errorf("signature config is nil")
	}
	log.Debugf("Verifying mail signature with method: %s", config.Method)

	switch config.Method {
	case SigningMethodRSA:
		valid, err = pwlib.RsaVerifyString(content, signature, config.PublicKeyFile)
	case SigningMethodECDSA:
		valid, err = pwlib.EcdsaVerifyString(content, signature, config.PublicKeyFile)
	case SigningMethodGPG:
		valid, err = verifySignatureWithGPG(content, signature, config)
	case SigningMethodSMIME:
		valid, err = verifySignatureWithSMIME(content, signature, config)
	default:
		err = fmt.Errorf("unsupported signing method: %s", config.Method)
	}

	if err != nil {
		log.Debugf("Verification failed: %v", err)
	}
	return
}

// signContentWithGPG signs content with GPG (wrapper around file-based signing)
func signContentWithGPG(content string, config *MailSignatureConfig) (signature string, err error) {
	log.Debugf("Signing with GPG using key: %s", config.PrivateKeyFile)

	// Create temp files
	tempDir := os.TempDir()
	contentFile, err := os.CreateTemp(tempDir, "mail_sign_*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp content file: %w", err)
	}
	tempContentFile := contentFile.Name()
	if err = contentFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp content file: %w", err)
	}
	defer func() { _ = os.Remove(tempContentFile) }()

	sigFile, err := os.CreateTemp(tempDir, "mail_sig_*.sig")
	if err != nil {
		return "", fmt.Errorf("failed to create temp signature file: %w", err)
	}
	tempSigFile := sigFile.Name()
	if err = sigFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp signature file: %w", err)
	}
	defer func() { _ = os.Remove(tempSigFile) }()

	// Write content to file
	if err = common.WriteStringToFile(tempContentFile, content); err != nil {
		return "", fmt.Errorf("failed to write content to temp file: %w", err)
	}

	// Setup GPG config
	pc := pwlib.NewConfig("mail_sign", "", "", "", "gpg")
	pc.PlainTextFile = tempContentFile
	pc.SignatureFile = tempSigFile
	pc.PrivateKeyFile = config.PrivateKeyFile
	pc.KeyPass = config.KeyPassphrase

	// Sign
	if err = pc.SignFile(); err != nil {
		return "", fmt.Errorf("GPG signing failed: %w", err)
	}

	// Read signature from file
	signature, err = common.ReadFileToString(tempSigFile)
	if err != nil {
		return "", fmt.Errorf("failed to read signature: %w", err)
	}

	log.Debug("GPG signing successful")
	return
}

// verifySignatureWithGPG verifies GPG signature
func verifySignatureWithGPG(content string, signature string, config *MailSignatureConfig) (valid bool, err error) {
	log.Debugf("Verifying GPG signature using key: %s", config.PublicKeyFile)

	// Create temp files
	tempDir := os.TempDir()
	contentFile, err := os.CreateTemp(tempDir, "mail_verify_*.txt")
	if err != nil {
		return false, fmt.Errorf("failed to create temp content file: %w", err)
	}
	tempContentFile := contentFile.Name()
	if err = contentFile.Close(); err != nil {
		return false, fmt.Errorf("failed to close temp content file: %w", err)
	}
	defer func() { _ = os.Remove(tempContentFile) }()

	sigFile, err := os.CreateTemp(tempDir, "mail_verify_sig_*.sig")
	if err != nil {
		return false, fmt.Errorf("failed to create temp signature file: %w", err)
	}
	tempSigFile := sigFile.Name()
	if err = sigFile.Close(); err != nil {
		return false, fmt.Errorf("failed to close temp signature file: %w", err)
	}
	defer func() { _ = os.Remove(tempSigFile) }()

	// Write files
	if err = common.WriteStringToFile(tempContentFile, content); err != nil {
		return false, fmt.Errorf("failed to write content: %w", err)
	}
	if err = common.WriteStringToFile(tempSigFile, signature); err != nil {
		return false, fmt.Errorf("failed to write signature: %w", err)
	}

	// Setup GPG config
	pc := pwlib.NewConfig("mail_verify", "", "", "", "gpg")
	pc.PlainTextFile = tempContentFile
	pc.SignatureFile = tempSigFile
	pc.PubKeyFile = config.PublicKeyFile

	// Verify
	valid, err = pc.VerifyFile()
	if err != nil {
		log.Debugf("GPG verification returned error: %v", err)
		return false, nil
	}
	return
}

// signContentWithSMIME signs content with S/MIME
func signContentWithSMIME(content string, config *MailSignatureConfig) (signature string, err error) {
	log.Debugf("Signing with S/MIME using certificate: %s", config.PrivateKeyFile)

	// Load certificate and private key
	privKey, cert, err := loadSMIMEPrivateKeyAndCert(config.PrivateKeyFile, config.KeyPassphrase)
	if err != nil {
		return "", fmt.Errorf("failed to load S/MIME certificate: %w", err)
	}

	// Create detached signature
	sigBytes, err := createDetachedSignature([]byte(content), privKey, cert)
	if err != nil {
		return "", fmt.Errorf("failed to create S/MIME signature: %w", err)
	}

	// Return signature as Base64
	signature = base64.StdEncoding.EncodeToString(sigBytes)
	log.Debug("S/MIME signing successful")
	return
}

// verifySignatureWithSMIME verifies S/MIME signature
func verifySignatureWithSMIME(content string, signature string, config *MailSignatureConfig) (valid bool, err error) {
	log.Debugf("Verifying S/MIME signature using certificate: %s", config.PublicKeyFile)
	certFile := config.PublicKeyFile
	if certFile == "" {
		certFile = config.PrivateKeyFile
	}

	// Load certificate
	cert, err := loadSMIMEPublicCert(certFile)
	if err != nil {
		return false, fmt.Errorf("failed to load S/MIME certificate: %w", err)
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify signature
	err = verifyDetachedSignature([]byte(content), sigBytes, cert)
	if err != nil {
		log.Debugf("S/MIME verification failed: %v", err)
		return false, nil
	}

	log.Debug("S/MIME verification successful")
	return true, nil
}

// Helper Functions for S/MIME

// decryptPEMKeyBytes returns the raw DER bytes of a PEM private-key block,
// decrypting with password when the block is encrypted.
func decryptPEMKeyBytes(block *pem.Block, password string) ([]byte, error) {
	//nolint:staticcheck
	if !x509.IsEncryptedPEMBlock(block) {
		return block.Bytes, nil
	}
	if password == "" {
		return nil, fmt.Errorf("encrypted private key requires password")
	}
	//nolint:staticcheck
	keyBytes, err := x509.DecryptPEMBlock(block, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}
	return keyBytes, nil
}

// parseRSAPrivateKeyBlock parses a PKCS#1 or PKCS#8 RSA private key PEM block.
func parseRSAPrivateKeyBlock(block *pem.Block, password string) (*rsa.PrivateKey, error) {
	keyBytes, err := decryptPEMKeyBytes(block, password)
	if err != nil {
		return nil, err
	}
	if block.Type == "RSA PRIVATE KEY" {
		return x509.ParsePKCS1PrivateKey(keyBytes)
	}
	priv, err := x509.ParsePKCS8PrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	privKey, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return privKey, nil
}

// loadSMIMEPrivateKeyAndCert loads private key and certificate from PEM file
func loadSMIMEPrivateKeyAndCert(certFile string, password string) (*rsa.PrivateKey, *x509.Certificate, error) {
	pemData, err := os.ReadFile(certFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	var privKey *rsa.PrivateKey
	var cert *x509.Certificate

	for len(pemData) > 0 {
		var block *pem.Block
		block, pemData = pem.Decode(pemData)
		if block == nil {
			break
		}
		switch block.Type {
		case "RSA PRIVATE KEY", "PRIVATE KEY":
			privKey, err = parseRSAPrivateKeyBlock(block, password)
			if err != nil {
				return nil, nil, err
			}
		case pemTypeCertificate:
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
			}
		}
	}

	if privKey == nil {
		return nil, nil, fmt.Errorf("private key not found in file")
	}
	return privKey, cert, nil
}

// loadSMIMEPublicCert loads certificate from PEM file for verification
func loadSMIMEPublicCert(certFile string) (*x509.Certificate, error) {
	pemData, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	var cert *x509.Certificate

	// Parse PEM blocks
	for len(pemData) > 0 {
		var block *pem.Block
		block, pemData = pem.Decode(pemData)
		if block == nil {
			break
		}

		if block.Type == pemTypeCertificate {
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse certificate: %w", err)
			}
			break
		}
	}

	if cert == nil {
		return nil, fmt.Errorf("no certificate found in file")
	}

	return cert, nil
}

// createDetachedSignature creates a detached S/MIME signature
func createDetachedSignature(content []byte, privKey *rsa.PrivateKey, cert *x509.Certificate) ([]byte, error) {
	if privKey == nil {
		return nil, fmt.Errorf("private key is nil")
	}
	if cert == nil {
		return nil, fmt.Errorf("certificate is nil")
	}
	return createSMIMEPKCS7DetachedSignature(content, cert, privKey, nil, false)
}

// verifyDetachedSignature verifies a detached S/MIME signature
func verifyDetachedSignature(content []byte, signature []byte, cert *x509.Certificate) error {
	valid, err := verifySMIMEPKCS7DetachedSignatureWithCert(content, signature, cert)
	if err != nil || !valid {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	return nil
}

// IsValidSigningMethod checks if a signing method is valid
func IsValidSigningMethod(method string) bool {
	validMethods := []SigningMethod{
		SigningMethodRSA,
		SigningMethodECDSA,
		SigningMethodGPG,
		SigningMethodSMIME,
	}

	for _, m := range validMethods {
		if string(m) == method {
			return true
		}
	}
	return false
}

// GetSupportedSigningMethods returns a list of supported signing methods
func GetSupportedSigningMethods() []string {
	return []string{
		string(SigningMethodRSA),
		string(SigningMethodECDSA),
		string(SigningMethodGPG),
		string(SigningMethodSMIME),
	}
}
