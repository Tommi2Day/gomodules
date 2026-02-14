package pwlib

import (
	"crypto/rsa"
	"os"
	"path"
	"testing"

	"github.com/tommi2day/gomodules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	p *rsa.PublicKey
	k *rsa.PrivateKey
)

func TestGenRsaKey(t *testing.T) {
	runKeyGenTest(t, "RSA", func(pub, priv, pass string) (any, any, error) {
		return GenRsaKey(pub, priv, pass)
	}, p, k)
}

func TestGetKeyFromFile(t *testing.T) {
	test.InitTestDirs()
	app := "test_get"
	testPubFile := path.Join(test.TestData, app+pubExt)
	testNotEncPrivFile := path.Join(test.TestData, app+"_notenc"+privExt)
	testEncPrivFile := path.Join(test.TestData, app+privExt)
	defaultPassword := app
	err := os.Chdir(test.TestDir)
	require.NoError(t, err, "ChDir failed")
	_ = os.Remove(testPubFile)
	_ = os.Remove(testNotEncPrivFile)
	_, _, err = GenRsaKey(testPubFile, testNotEncPrivFile, "")
	require.NoErrorf(t, err, "GenKey NoEncrypt failed failed:%s", err)
	t.Run("Get Public Key", func(t *testing.T) {
		pubkey, err := GetPublicKeyFromFile(testPubFile)
		assert.NoErrorf(t, err, "Error while reading pubkey: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.IsTypef(t, p, pubkey, "Not a public key")
	})
	t.Run("Get private key without password", func(t *testing.T) {
		pubkey, privkey, err := GetPrivateKeyFromFile(testNotEncPrivFile, "")
		assert.NoErrorf(t, err, "Error while reading privkey: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.IsTypef(t, p, pubkey, "Not a public key")
		assert.NotEmpty(t, privkey)
		assert.IsTypef(t, k, privkey, "Not a private key")
	})
	t.Run("Get private key with password, but should be none", func(t *testing.T) {
		pubkey, privkey, err := GetPrivateKeyFromFile(testNotEncPrivFile, defaultPassword)
		assert.Error(t, err, "Password given, but was not set")
		assert.Empty(t, pubkey)
		assert.Empty(t, privkey)
	})

	// test with encrypted passwords
	_ = os.Remove(testPubFile)
	_ = os.Remove(testEncPrivFile)
	_, _, err = GenRsaKey(testPubFile, testEncPrivFile, defaultPassword)
	require.NoErrorf(t, err, "GenKey NoEncrypt failed failed:%s", err)
	t.Run("Get private key with correct password", func(t *testing.T) {
		pubkey, privkey, err := GetPrivateKeyFromFile(testEncPrivFile, defaultPassword)
		assert.NoErrorf(t, err, "Error while reading privkey: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.IsTypef(t, p, pubkey, "Not a public key")
		assert.NotEmpty(t, privkey)
		assert.IsTypef(t, k, privkey, "Not a private key")
	})
	t.Run("Get private key with wrong password", func(t *testing.T) {
		pubkey, privkey, err := GetPrivateKeyFromFile(testEncPrivFile, "xxxx")
		assert.Errorf(t, err, "Wrong Password has been accepted: ")
		assert.Empty(t, pubkey)
		assert.Empty(t, privkey)
	})
	app = "test_pkcs1"
	t.Run("Get private key with PKCS1 (traditional openssl)", func(t *testing.T) {
		pubkey, privkey, err := GetPrivateKeyFromFile(app+".pem.txt", app)
		assert.NoErrorf(t, err, "Error while reading privkey: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.IsTypef(t, p, pubkey, "Not a public key")
		assert.NotEmpty(t, privkey)
		assert.IsTypef(t, k, privkey, "Not a private key")
	})
	t.Run("Get Public Key PKCS1", func(t *testing.T) {
		pubkey, err := GetPublicKeyFromFile(app + ".pub.txt")
		assert.NoErrorf(t, err, "Error while reading pubkey: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.IsTypef(t, p, pubkey, "Not a public key")
	})
}

func TestRsaSigningAndVerification(t *testing.T) {
	test.InitTestDirs()
	err := os.Chdir(test.TestDir)
	require.NoError(t, err, "ChDir failed")

	pubFile := "testdata/rsa_sign.pub"
	privFile := "testdata/rsa_sign.pem"
	password := "testpassword"
	message := "hello world"

	_ = os.Remove(pubFile)
	_ = os.Remove(privFile)

	_, _, err = GenRsaKey(pubFile, privFile, password)
	require.NoErrorf(t, err, "Failed to generate RSA key: %v", err)

	t.Run("Sign and Verify", func(t *testing.T) {
		signature, err := RsaSignString(message, privFile, password)
		assert.NoErrorf(t, err, "Failed to sign string: %v", err)
		assert.NotEmpty(t, signature)

		valid, err := RsaVerifyString(message, signature, pubFile)
		assert.NoErrorf(t, err, "Failed to verify signature: %v", err)
		assert.True(t, valid, "Signature verification failed")

		// Test with wrong message
		valid, err = RsaVerifyString("wrong message", signature, pubFile)
		assert.NoErrorf(t, err, "Failed to verify signature with wrong message: %v", err)
		assert.False(t, valid, "Signature verification should have failed for wrong message")
	})
}
