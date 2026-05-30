package maillib

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/pwlib"
	"github.com/tommi2day/gomodules/test"
)

func TestMailSigningRSA(t *testing.T) {
	test.InitTestDirs()

	t.Run("Sign and Verify Mail - RSA", func(t *testing.T) {
		// Generate RSA keys
		pubKeyFile := test.TestData + "/test_mail_rsa.pub"
		privKeyFile := test.TestData + "/test_mail_rsa.key"
		keyPass := "testpass"

		_, _, err := pwlib.GenRsaKey(pubKeyFile, privKeyFile, keyPass)
		require.NoError(t, err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		// Create mail
		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"This is a test message to sign"}

		// Sign mail
		config := &MailSignatureConfig{
			Method:         SigningMethodRSA,
			PrivateKeyFile: privKeyFile,
			PublicKeyFile:  pubKeyFile,
			KeyPassphrase:  keyPass,
		}

		err = mail.SignMail(config)
		assert.NoError(t, err)
		assert.True(t, mail.IsSigned)
		assert.NotEmpty(t, mail.Signature)

		// Verify signature
		valid, err := mail.VerifyMailSignature()
		assert.NoError(t, err)
		assert.True(t, valid)
		assert.True(t, mail.SignatureVerified)
	})

	t.Run("Verify Mail - Invalid RSA Signature", func(t *testing.T) {
		pubKeyFile := test.TestData + "/test_mail_rsa_invalid.pub"
		privKeyFile := test.TestData + "/test_mail_rsa_invalid.key"

		_, _, err := pwlib.GenRsaKey(pubKeyFile, privKeyFile, "testpass")
		require.NoError(t, err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"Original message"}
		mail.Signature = "invalid_signature_data"
		mail.IsSigned = true
		mail.SignatureConfig = &MailSignatureConfig{
			Method:        SigningMethodRSA,
			PublicKeyFile: pubKeyFile,
		}

		valid, err := mail.VerifyMailSignature()
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("Sign Mail - RSA with Multiple Text Parts", func(t *testing.T) {
		pubKeyFile := test.TestData + "/test_mail_rsa_multi.pub"
		privKeyFile := test.TestData + "/test_mail_rsa_multi.key"

		_, _, err := pwlib.GenRsaKey(pubKeyFile, privKeyFile, "testpass")
		require.NoError(t, err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"Part 1", "Part 2", "Part 3"}

		config := &MailSignatureConfig{
			Method:         SigningMethodRSA,
			PrivateKeyFile: privKeyFile,
			PublicKeyFile:  pubKeyFile,
			KeyPassphrase:  "testpass",
		}

		err = mail.SignMail(config)
		assert.NoError(t, err)

		valid, err := mail.VerifyMailSignature()
		assert.NoError(t, err)
		assert.True(t, valid)
	})
}

func TestMailSigningECDSA(t *testing.T) {
	test.InitTestDirs()

	t.Run("Sign and Verify Mail - ECDSA", func(t *testing.T) {
		// Generate ECDSA keys
		pubKeyFile := test.TestData + "/test_mail_ecdsa.pub"
		privKeyFile := test.TestData + "/test_mail_ecdsa.key"
		keyPass := "testpass"

		_, _, err := pwlib.GenEcdsaKey(pubKeyFile, privKeyFile, keyPass)
		require.NoError(t, err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		// Create mail
		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"ECDSA signed test message"}

		// Sign mail
		config := &MailSignatureConfig{
			Method:         SigningMethodECDSA,
			PrivateKeyFile: privKeyFile,
			PublicKeyFile:  pubKeyFile,
			KeyPassphrase:  keyPass,
		}

		err = mail.SignMail(config)
		assert.NoError(t, err)
		assert.True(t, mail.IsSigned)

		// Verify
		valid, err := mail.VerifyMailSignature()
		assert.NoError(t, err)
		assert.True(t, valid)
	})
}

func TestMailSigningGPG(t *testing.T) {
	if os.Getenv("SKIP_GPG") != "" {
		t.Skip("Skipping GPG test")
	}

	test.InitTestDirs()

	t.Run("Sign and Verify Mail - GPG", func(t *testing.T) {
		// Setup GPG keys
		pubKeyFile := test.TestData + "/test_mail_gpg.pub"
		privKeyFile := test.TestData + "/test_mail_gpg.priv"

		entity, _, err := pwlib.CreateGPGEntity("Mail Test", "mail", "mail@test.com", "testpass")
		require.NoError(t, err)

		err = pwlib.ExportGPGKeyPair(entity, pubKeyFile, privKeyFile)
		require.NoError(t, err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		// Create and sign mail
		mail := NewMail("mail@test.com", "recipient@test.com")
		mail.TextParts = []string{"GPG signed test message"}

		config := &MailSignatureConfig{
			Method:         SigningMethodGPG,
			PrivateKeyFile: privKeyFile,
			PublicKeyFile:  pubKeyFile,
			KeyPassphrase:  "testpass",
		}

		err = mail.SignMail(config)
		assert.NoError(t, err)
		assert.True(t, mail.IsSigned)

		// Verify
		valid, err := mail.VerifyMailSignature()
		assert.NoError(t, err)
		assert.True(t, valid)
	})
}

func TestMailSigningSMIME(t *testing.T) {
	test.InitTestDirs()

	t.Run("Sign and Verify Mail - S/MIME", func(t *testing.T) {
		bundleFile := filepath.Join(test.TestData, "test_mail_smime_bundle.pem")
		certFile := filepath.Join(test.TestData, "test_mail_smime_cert.pem")
		err := createSMIMEFixture(bundleFile, certFile)
		require.NoError(t, err)
		defer os.Remove(bundleFile)
		defer os.Remove(certFile)

		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"S/MIME signed test message"}

		config := &MailSignatureConfig{
			Method:         SigningMethodSMIME,
			PrivateKeyFile: bundleFile,
			PublicKeyFile:  certFile,
		}

		err = mail.SignMail(config)
		require.NoError(t, err)
		require.NotEmpty(t, mail.Signature)

		valid, err := mail.VerifyMailSignature()
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Verify Mail - S/MIME Tampered Content", func(t *testing.T) {
		bundleFile := filepath.Join(test.TestData, "test_mail_smime_bundle_tampered.pem")
		certFile := filepath.Join(test.TestData, "test_mail_smime_cert_tampered.pem")
		err := createSMIMEFixture(bundleFile, certFile)
		require.NoError(t, err)
		defer os.Remove(bundleFile)
		defer os.Remove(certFile)

		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"Original signed content"}
		mail.SignatureConfig = &MailSignatureConfig{
			Method:         SigningMethodSMIME,
			PrivateKeyFile: bundleFile,
			PublicKeyFile:  certFile,
		}

		err = mail.SignMail(mail.SignatureConfig)
		require.NoError(t, err)

		// Tamper after signing.
		mail.TextParts = []string{"Tampered content"}
		valid, err := mail.VerifyMailSignature()
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("Build and Verify multipart/signed S/MIME", func(t *testing.T) {
		bundleFile := filepath.Join(test.TestData, "test_mail_smime_bundle_mime.pem")
		certFile := filepath.Join(test.TestData, "test_mail_smime_cert_mime.pem")
		err := createSMIMEFixture(bundleFile, certFile)
		require.NoError(t, err)
		defer os.Remove(bundleFile)
		defer os.Remove(certFile)

		cfg := &MailSignatureConfig{
			Method:         SigningMethodSMIME,
			PrivateKeyFile: bundleFile,
			PublicKeyFile:  certFile,
		}

		mimeBody, contentType, signature, err := BuildSMIMEMultipartSigned("Line1\nLine2", cfg)
		require.NoError(t, err)
		require.NotEmpty(t, signature)
		assert.Contains(t, strings.ToLower(contentType), "multipart/signed")
		assert.Contains(t, strings.ToLower(contentType), "application/pkcs7-signature")

		signedContent, valid, err := VerifySMIMEMultipartSigned(mimeBody, contentType)
		require.NoError(t, err)
		assert.True(t, valid)
		assert.Contains(t, signedContent, "Line1")
		assert.Contains(t, signedContent, "Line2")

		tampered := strings.Replace(mimeBody, "Line2", "Line2-TAMPERED", 1)
		_, valid, err = VerifySMIMEMultipartSigned(tampered, contentType)
		require.NoError(t, err)
		assert.False(t, valid)
	})
}

func createSMIMEFixture(bundleFile, certFile string) error {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "mail signer",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &privKey.PublicKey, privKey)
	if err != nil {
		return err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)})

	bundle := make([]byte, 0, len(certPEM)+len(privPEM))
	bundle = append(bundle, certPEM...)
	bundle = append(bundle, privPEM...)
	if err := os.WriteFile(bundleFile, bundle, 0o600); err != nil {
		return err
	}
	return os.WriteFile(certFile, certPEM, 0o600)
}

func TestMailSigningErrors(t *testing.T) {
	t.Run("Sign Mail - Unsupported Method", func(t *testing.T) {
		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"Test"}

		config := &MailSignatureConfig{
			Method: SigningMethod("invalid"),
		}

		err := mail.SignMail(config)
		assert.Error(t, err)
	})

	t.Run("Sign Mail - Nil Config", func(t *testing.T) {
		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"Test"}

		err := mail.SignMail(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config is nil")
	})

	t.Run("Sign Mail - No Content", func(t *testing.T) {
		mail := NewMail("sender@example.com", "recipient@example.com")

		config := &MailSignatureConfig{
			Method: SigningMethodRSA,
		}

		err := mail.SignMail(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no content to sign")
	})

	t.Run("Verify Mail - Not Signed", func(t *testing.T) {
		mail := NewMail("sender@example.com", "recipient@example.com")

		valid, err := mail.VerifyMailSignature()
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "is not signed")
	})

	t.Run("Verify Mail - Missing Config", func(t *testing.T) {
		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.IsSigned = true
		mail.Signature = "somesignature"

		valid, err := mail.VerifyMailSignature()
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "config not set")
	})
}

func TestMailSigningHelpers(t *testing.T) {
	t.Run("IsValidSigningMethod - Valid Methods", func(t *testing.T) {
		assert.True(t, IsValidSigningMethod("rsa"))
		assert.True(t, IsValidSigningMethod("ecdsa"))
		assert.True(t, IsValidSigningMethod("gpg"))
		assert.True(t, IsValidSigningMethod("smime"))
	})

	t.Run("IsValidSigningMethod - Invalid Methods", func(t *testing.T) {
		assert.False(t, IsValidSigningMethod("invalid"))
		assert.False(t, IsValidSigningMethod(""))
		assert.False(t, IsValidSigningMethod("dsa"))
	})

	t.Run("GetSupportedSigningMethods", func(t *testing.T) {
		methods := GetSupportedSigningMethods()
		assert.NotEmpty(t, methods)
		assert.Equal(t, 4, len(methods))
		assert.Contains(t, methods, "rsa")
		assert.Contains(t, methods, "ecdsa")
		assert.Contains(t, methods, "gpg")
		assert.Contains(t, methods, "smime")
	})

	t.Run("GetSignatureMethod", func(t *testing.T) {
		mail := NewMail("sender@example.com", "recipient@example.com")

		// No signature set
		assert.Empty(t, mail.GetSignatureMethod())

		// With RSA signature
		mail.SignatureConfig = &MailSignatureConfig{
			Method: SigningMethodRSA,
		}
		assert.Equal(t, "rsa", mail.GetSignatureMethod())
	})
}

func TestMailSigningIntegration(t *testing.T) {
	test.InitTestDirs()

	t.Run("Complete Mail Signing Workflow - RSA", func(t *testing.T) {
		// Setup
		pubKeyFile := test.TestData + "/test_workflow_rsa.pub"
		privKeyFile := test.TestData + "/test_workflow_rsa.key"

		_, _, err := pwlib.GenRsaKey(pubKeyFile, privKeyFile, "pass123")
		require.NoError(t, err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		// Create complete mail
		mail := NewMail("john@example.com", "jane@example.com")
		mail.Subject = "Signed Message"
		mail.TextParts = []string{
			"Hello Jane,",
			"",
			"This is a signed message.",
			"Best regards,",
			"John",
		}

		// Sign
		signConfig := &MailSignatureConfig{
			Method:         SigningMethodRSA,
			PrivateKeyFile: privKeyFile,
			PublicKeyFile:  pubKeyFile,
			KeyPassphrase:  "pass123",
		}

		err = mail.SignMail(signConfig)
		require.NoError(t, err)
		require.True(t, mail.IsSigned)

		// Simulate sending and receiving (signature stays the same)
		receivedMail := &MailType{
			From:              mail.From,
			Subject:           mail.Subject,
			TextParts:         mail.TextParts,
			Signature:         mail.Signature,
			SignatureConfig:   mail.SignatureConfig,
			IsSigned:          true,
			SignatureVerified: false,
		}

		// Verify
		valid, err := receivedMail.VerifyMailSignature()
		require.NoError(t, err)
		assert.True(t, valid)
		assert.True(t, receivedMail.SignatureVerified)
	})
}

func TestSigningMethodConstants(t *testing.T) {
	t.Run("Signing Method Constants", func(t *testing.T) {
		assert.Equal(t, SigningMethod("rsa"), SigningMethodRSA)
		assert.Equal(t, SigningMethod("ecdsa"), SigningMethodECDSA)
		assert.Equal(t, SigningMethod("gpg"), SigningMethodGPG)
		assert.Equal(t, SigningMethod("smime"), SigningMethodSMIME)
	})
}

func TestMailSigningCrossKey(t *testing.T) {
	test.InitTestDirs()

	t.Run("RSA - cross-key verification failure", func(t *testing.T) {
		pubKeyA := test.TestData + "/test_cross_rsa_a.pub"
		privKeyA := test.TestData + "/test_cross_rsa_a.key"
		pubKeyB := test.TestData + "/test_cross_rsa_b.pub"
		privKeyB := test.TestData + "/test_cross_rsa_b.key"

		_, _, err := pwlib.GenRsaKey(pubKeyA, privKeyA, "pass")
		require.NoError(t, err)
		defer os.Remove(pubKeyA)
		defer os.Remove(privKeyA)

		_, _, err = pwlib.GenRsaKey(pubKeyB, privKeyB, "pass")
		require.NoError(t, err)
		defer os.Remove(pubKeyB)
		defer os.Remove(privKeyB)

		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"Test message"}

		err = mail.SignMail(&MailSignatureConfig{
			Method:         SigningMethodRSA,
			PrivateKeyFile: privKeyA,
			PublicKeyFile:  pubKeyA,
			KeyPassphrase:  "pass",
		})
		require.NoError(t, err)

		// Swap to key B for verification — must fail
		mail.SignatureConfig = &MailSignatureConfig{
			Method:        SigningMethodRSA,
			PublicKeyFile: pubKeyB,
		}
		valid, err := mail.VerifyMailSignature()
		// RsaVerifyString returns (false, nil) on key mismatch
		assert.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("ECDSA - invalid signature", func(t *testing.T) {
		pubKeyFile := test.TestData + "/test_ecdsa_invalid.pub"
		privKeyFile := test.TestData + "/test_ecdsa_invalid.key"
		_, _, err := pwlib.GenEcdsaKey(pubKeyFile, privKeyFile, "pass")
		require.NoError(t, err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		mail := NewMail("sender@example.com", "recipient@example.com")
		mail.TextParts = []string{"Original message"}
		mail.Signature = "invalidsignaturedata"
		mail.IsSigned = true
		mail.SignatureConfig = &MailSignatureConfig{
			Method:        SigningMethodECDSA,
			PublicKeyFile: pubKeyFile,
		}

		valid, err := mail.VerifyMailSignature()
		assert.Error(t, err)
		assert.False(t, valid)
	})
}

func TestSMIMEMultipartErrors(t *testing.T) {
	test.InitTestDirs()

	t.Run("BuildSMIMEMultipartSigned - nil config", func(t *testing.T) {
		_, _, _, err := BuildSMIMEMultipartSigned("content", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config is nil")
	})

	t.Run("BuildSMIMEMultipartSigned - wrong method", func(t *testing.T) {
		_, _, _, err := BuildSMIMEMultipartSigned("content", &MailSignatureConfig{Method: SigningMethodRSA})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signing method must be smime")
	})

	t.Run("VerifySMIMEMultipartSigned - invalid content-type", func(t *testing.T) {
		_, valid, err := VerifySMIMEMultipartSigned("body", "not///a-valid-content-type")
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("VerifySMIMEMultipartSigned - non-multipart/signed", func(t *testing.T) {
		_, valid, err := VerifySMIMEMultipartSigned("body", "text/plain")
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "not multipart/signed")
	})

	t.Run("VerifySMIMEMultipartSigned - wrong protocol", func(t *testing.T) {
		_, valid, err := VerifySMIMEMultipartSigned("body", `multipart/signed; protocol="application/pgp-signature"; boundary="bound"`)
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "unsupported multipart/signed protocol")
	})

	t.Run("VerifySMIMEMultipartSigned - missing boundary", func(t *testing.T) {
		_, valid, err := VerifySMIMEMultipartSigned("body", `multipart/signed; protocol="application/pkcs7-signature"`)
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "missing multipart boundary")
	})

	t.Run("VerifySMIMEMultipartSigned - invalid base64 signature", func(t *testing.T) {
		body := "--bound\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 7bit\r\n\r\nhello\r\n" +
			"--bound\r\nContent-Type: application/pkcs7-signature; name=\"smime.p7s\"\r\nContent-Transfer-Encoding: base64\r\n\r\n" +
			"!!!NOT_VALID_BASE64!!!\r\n--bound--\r\n"
		ct := `multipart/signed; protocol="application/pkcs7-signature"; micalg=sha-256; boundary="bound"`
		_, valid, err := VerifySMIMEMultipartSigned(body, ct)
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("VerifySMIMEMultipartSigned - tampered signature bytes", func(t *testing.T) {
		bundleFile := filepath.Join(test.TestData, "test_smime_tamsig_bundle.pem")
		certFile := filepath.Join(test.TestData, "test_smime_tamsig_cert.pem")
		require.NoError(t, createSMIMEFixture(bundleFile, certFile))
		defer os.Remove(bundleFile)
		defer os.Remove(certFile)

		cfg := &MailSignatureConfig{
			Method:         SigningMethodSMIME,
			PrivateKeyFile: bundleFile,
			PublicKeyFile:  certFile,
		}
		body, contentType, _, err := BuildSMIMEMultipartSigned("tamper-sig test", cfg)
		require.NoError(t, err)

		// Replace the last character of the body before the final boundary
		// to corrupt the base64-encoded signature.
		endMarker := "--smime-"
		idx := strings.LastIndex(body, endMarker)
		if idx > 10 {
			body = body[:idx-2] + "XX" + body[idx:]
		}
		_, valid, err := VerifySMIMEMultipartSigned(body, contentType)
		// Corrupted signature must not verify as valid.
		if err == nil {
			assert.False(t, valid)
		}
	})
}

func TestSMIMECertChain(t *testing.T) {
	test.InitTestDirs()

	t.Run("BuildSMIMEMultipartSigned - with cert chain", func(t *testing.T) {
		bundleFile := filepath.Join(test.TestData, "test_smime_chain_bundle.pem")
		certFile := filepath.Join(test.TestData, "test_smime_chain_cert.pem")
		require.NoError(t, createSMIMEFixture(bundleFile, certFile))
		defer os.Remove(bundleFile)
		defer os.Remove(certFile)

		cfg := &MailSignatureConfig{
			Method:           SigningMethodSMIME,
			PrivateKeyFile:   bundleFile,
			PublicKeyFile:    certFile,
			CertificateChain: []string{certFile},
			IncludeChain:     true,
		}

		body, contentType, sig, err := BuildSMIMEMultipartSigned("chain test content", cfg)
		require.NoError(t, err)
		assert.NotEmpty(t, sig)

		signedContent, valid, err := VerifySMIMEMultipartSigned(body, contentType)
		require.NoError(t, err)
		assert.True(t, valid)
		assert.Contains(t, signedContent, "chain test content")
	})

	t.Run("S/MIME - cert mismatch during verification", func(t *testing.T) {
		bundleA := filepath.Join(test.TestData, "test_smime_mismatch_bundle_a.pem")
		certA := filepath.Join(test.TestData, "test_smime_mismatch_cert_a.pem")
		bundleB := filepath.Join(test.TestData, "test_smime_mismatch_bundle_b.pem")
		certB := filepath.Join(test.TestData, "test_smime_mismatch_cert_b.pem")

		require.NoError(t, createSMIMEFixture(bundleA, certA))
		require.NoError(t, createSMIMEFixture(bundleB, certB))
		defer func() {
			os.Remove(bundleA)
			os.Remove(certA)
			os.Remove(bundleB)
			os.Remove(certB)
		}()

		mail := NewMail("s@example.com", "r@example.com")
		mail.TextParts = []string{"cert mismatch test"}

		// Sign with cert A, verify expecting cert B
		err := mail.SignMail(&MailSignatureConfig{
			Method:         SigningMethodSMIME,
			PrivateKeyFile: bundleA,
			PublicKeyFile:  certA,
		})
		require.NoError(t, err)

		mail.SignatureConfig.PublicKeyFile = certB
		valid, err := mail.VerifyMailSignature()
		assert.False(t, valid)
		// err may or may not be set depending on how the pkcs7 verify surfaces the mismatch
		_ = err
	})

	t.Run("S/MIME - cert-only file missing private key", func(t *testing.T) {
		privKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(42),
			Subject:      pkix.Name{CommonName: "cert-only"},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(24 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &privKey.PublicKey, privKey)
		require.NoError(t, err)

		certOnlyFile := filepath.Join(test.TestData, "test_smime_certonly.pem")
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		require.NoError(t, os.WriteFile(certOnlyFile, certPEM, 0o600))
		defer os.Remove(certOnlyFile)

		mail := NewMail("s@example.com", "r@example.com")
		mail.TextParts = []string{"no private key"}
		err = mail.SignMail(&MailSignatureConfig{
			Method:         SigningMethodSMIME,
			PrivateKeyFile: certOnlyFile,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "private key not found")
	})
}
