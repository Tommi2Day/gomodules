package pwlib

import (
	"crypto/ecdsa"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/test"
)

var (
	ep *ecdsa.PublicKey
	ek *ecdsa.PrivateKey
)

func TestGenEcdsaKey(t *testing.T) {
	runKeyGenTest(t, "ECDSA", func(pub, priv, pass string) (any, any, error) {
		return GenEcdsaKey(pub, priv, pass)
	}, ep, ek)
}

func TestGetEcdsaKeyFromFile(t *testing.T) {
	test.InitTestDirs()
	app := "test_ecdsa_get"
	testPubFile := path.Join(test.TestData, app+pubExt)
	testNotEncPrivFile := path.Join(test.TestData, app+"_notenc"+privExt)
	testEncPrivFile := path.Join(test.TestData, app+privExt)
	defaultPassword := app
	err := os.Chdir(test.TestDir)
	require.NoError(t, err, "ChDir failed")

	_ = os.Remove(testPubFile)
	_ = os.Remove(testNotEncPrivFile)
	_, _, err = GenEcdsaKey(testPubFile, testNotEncPrivFile, "")
	require.NoErrorf(t, err, "GenEcdsaKey NoEncrypt failed:%s", err)

	t.Run("Get ECDSA Public Key", func(t *testing.T) {
		pubkey, err := GetEcdsaPublicKeyFromFile(testPubFile)
		assert.NoErrorf(t, err, "Error while reading pubkey: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.IsTypef(t, ep, pubkey, "Not an ECDSA public key")
	})

	t.Run("Get ECDSA private key without password", func(t *testing.T) {
		pubkey, privkey, err := GetEcdsaPrivateKeyFromFile(testNotEncPrivFile, "")
		assert.NoErrorf(t, err, "Error while reading privkey: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.IsTypef(t, ep, pubkey, "Not an ECDSA public key")
		assert.NotEmpty(t, privkey)
		assert.IsTypef(t, ek, privkey, "Not an ECDSA private key")
	})

	t.Run("Get ECDSA private key with password, but should be none", func(t *testing.T) {
		pubkey, privkey, err := GetEcdsaPrivateKeyFromFile(testNotEncPrivFile, defaultPassword)
		assert.Error(t, err, "Password given, but was not set")
		assert.Empty(t, pubkey)
		assert.Empty(t, privkey)
	})

	// test with encrypted passwords
	_ = os.Remove(testPubFile)
	_ = os.Remove(testEncPrivFile)
	_, _, err = GenEcdsaKey(testPubFile, testEncPrivFile, defaultPassword)
	require.NoErrorf(t, err, "GenEcdsaKey Encrypted failed:%s", err)

	t.Run("Get ECDSA private key with correct password", func(t *testing.T) {
		pubkey, privkey, err := GetEcdsaPrivateKeyFromFile(testEncPrivFile, defaultPassword)
		assert.NoErrorf(t, err, "Error while reading privkey: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.IsTypef(t, ep, pubkey, "Not an ECDSA public key")
		assert.NotEmpty(t, privkey)
		assert.IsTypef(t, ek, privkey, "Not an ECDSA private key")
	})

	t.Run("Get ECDSA private key with wrong password", func(t *testing.T) {
		pubkey, privkey, err := GetEcdsaPrivateKeyFromFile(testEncPrivFile, "xxxx")
		assert.Errorf(t, err, "Wrong Password has been accepted")
		assert.Empty(t, pubkey)
		assert.Empty(t, privkey)
	})
}

func TestEcdsaSigningAndVerification(t *testing.T) {
	test.InitTestDirs()
	err := os.Chdir(test.TestDir)
	require.NoError(t, err, "ChDir failed")

	pubFile := "testdata/ecdsa_sign.pub"
	privFile := "testdata/ecdsa_sign.pem"
	password := "testpassword"
	message := "hello world"

	_ = os.Remove(pubFile)
	_ = os.Remove(privFile)

	_, _, err = GenEcdsaKey(pubFile, privFile, password)
	require.NoErrorf(t, err, "Failed to generate ECDSA key: %v", err)

	t.Run("Sign and Verify", func(t *testing.T) {
		signature, err := EcdsaSignString(message, privFile, password)
		assert.NoErrorf(t, err, "Failed to sign string: %v", err)
		assert.NotEmpty(t, signature)

		valid, err := EcdsaVerifyString(message, signature, pubFile)
		assert.NoErrorf(t, err, "Failed to verify signature: %v", err)
		assert.True(t, valid, "Signature verification failed")

		// Test with wrong message
		valid, err = EcdsaVerifyString("wrong message", signature, pubFile)
		assert.NoErrorf(t, err, "Failed to verify signature with wrong message: %v", err)
		assert.False(t, valid, "Signature verification should have failed for wrong message")
	})
}
