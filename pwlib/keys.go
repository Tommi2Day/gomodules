package pwlib

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
)

// PrivateDecryptString decrypts a string using a private key (RSA)
func PrivateDecryptString(crypted string, privateKeyFile string, keyPass string) (plain string, err error) {
	keyType, err := GetKeyTypeFromFile(privateKeyFile)
	if err != nil {
		return "", err
	}

	switch keyType {
	case KeyTypeRSA:
		return RsaDecryptString(crypted, privateKeyFile, keyPass)
	case KeyTypeECDSA:
		return "", fmt.Errorf("decryption not supported for ECDSA keys")
	default:
		return "", fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// PublicEncryptString encrypts a string using a public key (RSA)
func PublicEncryptString(plain string, publicKeyFile string) (crypted string, err error) {
	keyType, err := GetKeyTypeFromFile(publicKeyFile)
	if err != nil {
		return "", err
	}

	switch keyType {
	case KeyTypeRSA:
		return RsaEncryptString(plain, publicKeyFile)
	case KeyTypeECDSA:
		return "", fmt.Errorf("encryption not supported for ECDSA keys")
	default:
		return "", fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// SignString signs a string using a private key (RSA or ECDSA)
func SignString(plain string, privateKeyFile string, keyPass string) (signature string, err error) {
	keyType, err := GetKeyTypeFromFile(privateKeyFile)
	if err != nil {
		return "", err
	}

	switch keyType {
	case KeyTypeRSA:
		return RsaSignString(plain, privateKeyFile, keyPass)
	case KeyTypeECDSA:
		return EcdsaSignString(plain, privateKeyFile, keyPass)
	default:
		return "", fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// VerifyString verifies a string signature using a public key (RSA or ECDSA)
func VerifyString(plain string, signature string, publicKeyFile string) (valid bool, err error) {
	keyType, err := GetKeyTypeFromFile(publicKeyFile)
	if err != nil {
		return false, err
	}

	switch keyType {
	case KeyTypeRSA:
		return RsaVerifyString(plain, signature, publicKeyFile)
	case KeyTypeECDSA:
		return EcdsaVerifyString(plain, signature, publicKeyFile)
	default:
		return false, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// GetKeyTypeFromFile detects the type of key in a PEM file
func GetKeyTypeFromFile(keyFile string) (keyType string, err error) {
	log.Debugf("GetKeyTypeFromFile entered for %s", keyFile)
	data, err := common.ReadFileToString(keyFile)
	if err != nil {
		log.Debugf("cannot read %s: %s", keyFile, err)
		return KeyTypeUnknown, err
	}

	block, _ := pem.Decode([]byte(data))
	if block == nil {
		// If not PEM, check for GPG or AGE
		// AGE recipient (public key) starts with "age1"
		if strings.HasPrefix(data, "age1") {
			return KeyTypeAGE, nil
		}
		if strings.HasPrefix(data, "age-encryption.org/v1") {
			return KeyTypeAGE, nil
		}
		if strings.Contains(data, "AGE-SECRET-KEY") {
			return KeyTypeAGE, nil
		}
		// GPG binary check (simple check for GPG packet header if possible, but PGP/GPG usually starts with 0x85, 0x89 etc)
		// Better to check for "PGP" in armored or generic PGP headers
		if strings.Contains(data, "-----BEGIN PGP") {
			return KeyTypeGPG, nil
		}
		log.Debugf("cannot decode pem in %s", keyFile)
		return KeyTypeUnknown, fmt.Errorf("cannot decode pem in %s", keyFile)
	}

	log.Debugf("PEM Type: %s", block.Type)

	// Identify by PEM Type first
	switch block.Type {
	case "RSA PRIVATE KEY", "RSA PUBLIC KEY":
		return KeyTypeRSA, nil
	case "EC PRIVATE KEY", "EC PUBLIC KEY":
		return KeyTypeECDSA, nil
	case "PGP PRIVATE KEY BLOCK", "PGP PUBLIC KEY BLOCK", "PGP MESSAGE":
		return KeyTypeGPG, nil
	case "PUBLIC KEY", "PRIVATE KEY":
		// These are generic PKCS8 or PKIX types, need to parse bytes
		return detectKeyTypeFromBytes(block.Bytes)
	}

	// Fallback/Legacy checks if Type is something else but might still be a key
	if strings.Contains(block.Type, "RSA") {
		return KeyTypeRSA, nil
	}
	if strings.Contains(block.Type, "EC") || strings.Contains(block.Type, "ECDSA") {
		return KeyTypeECDSA, nil
	}
	if strings.Contains(block.Type, "PGP") {
		return KeyTypeGPG, nil
	}

	return KeyTypeUnknown, nil
}

func detectKeyTypeFromBytes(der []byte) (string, error) {
	// Try public key first
	if pub, err := x509.ParsePKIXPublicKey(der); err == nil {
		switch pub.(type) {
		case *rsa.PublicKey:
			return KeyTypeRSA, nil
		case *ecdsa.PublicKey:
			return KeyTypeECDSA, nil
		}
	}

	// Try private key (PKCS8)
	if priv, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch priv.(type) {
		case *rsa.PrivateKey:
			return KeyTypeRSA, nil
		case *ecdsa.PrivateKey:
			return KeyTypeECDSA, nil
		}
	}

	// Try RSA PKCS1
	if _, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return KeyTypeRSA, nil
	}

	// Try EC SEC1
	if _, err := x509.ParseECPrivateKey(der); err == nil {
		return KeyTypeECDSA, nil
	}

	return KeyTypeUnknown, nil
}
