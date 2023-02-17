package pwlib

import (
	"crypto/rsa"
	"github.com/tommi2day/gomodules/test"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	p *rsa.PublicKey
	k *rsa.PrivateKey
)

func TestGenRsaKey(t *testing.T) {
	err := os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")

	pubfilename := "testdata/key.pub"
	privfilename := "testdata/key.pem"
	_ = os.Remove(pubfilename)
	_ = os.Remove(privfilename)

	t.Run("Key Gen unencrypted", func(t *testing.T) {
		pubkey, privkey, err := GenRsaKey(pubfilename, privfilename, "")
		assert.NoErrorf(t, err, "Error while creating key: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.NotEmpty(t, privkey)
		assert.IsTypef(t, p, pubkey, "Not a public key")
		assert.IsTypef(t, k, privkey, "Not a private key")
		assert.FileExists(t, pubfilename)
		assert.FileExists(t, privfilename)
	})
	pubfilename = "testdata/enckey.pub"
	privfilename = "testdata/enckey.pem"
	_ = os.Remove(pubfilename)
	_ = os.Remove(privfilename)
	t.Run("Key Gen encrypted", func(t *testing.T) {
		pubkey, privkey, err := GenRsaKey(pubfilename, privfilename, "gen_test")
		assert.NoErrorf(t, err, "Error while creating key: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.NotEmpty(t, privkey)
		assert.IsTypef(t, p, pubkey, "Not a public key")
		assert.IsTypef(t, k, privkey, "Not a private key")
		assert.FileExists(t, pubfilename)
		assert.FileExists(t, privfilename)
		//nolint gosec
		content, err := os.ReadFile(privfilename)
		assert.NoErrorf(t, err, "File Read Error %s", err)
		assert.Contains(t, string(content), "Proc-Type: 4,ENCRYPTED")
	})
}

func TestGetKeyFromFile(t *testing.T) {
	app := "test_get"
	testPubFile := "testdata/" + app + ".pub"
	testNotEncPrivFile := "testdata/" + app + "_notenc.pem"
	testEncPrivFile := "testdata/" + app + ".pem"
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
}
