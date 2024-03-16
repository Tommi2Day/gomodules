package pwlib

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"

	"github.com/ProtonMail/go-crypto/openpgp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testGPGName = "Test User"
const testGPGEmail = "test@example.com"
const testGPGPass = "123456Pass!"

func TestGPG(t *testing.T) {
	var err error
	var gpgid string
	var keypass string
	var key string
	var entityList openpgp.EntityList
	var entity *openpgp.Entity
	test.Testinit(t)

	secretGPGKeyFile := path.Join(test.TestData, "test.gpg.key")
	publicGPGKeyFile := path.Join(test.TestData, "test.asc")
	_ = os.Remove(publicGPGKeyFile)
	_ = os.Remove(secretGPGKeyFile)

	t.Run("GPG Gen Key", func(t *testing.T) {
		keypass = testGPGPass
		entity, gpgid, err = CreateGPGEntity(testGPGName, "TestCrypt", testGPGEmail, keypass)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		assert.NotNil(t, entity, "entity should not be nil")
		if entity == nil {
			t.Fatal("entity should not be nil")
		}
		assert.NotEmpty(t, gpgid, "KeyID should not be empty")
		assert.True(t, entity.PrivateKey.Encrypted, "encrypted flag should be true")
		err = ExportGPGKeyPair(entity, publicGPGKeyFile, secretGPGKeyFile)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		if err != nil {
			t.Fatal("GPG keys not created as expected")
		}
		require.FileExists(t, publicGPGKeyFile)
		content := ""
		content, err = common.ReadFileToString(publicGPGKeyFile)
		assert.NoErrorf(t, err, "File Read Error %s", err)
		assert.Contains(t, content, "PGP PUBLIC KEY BLOCK")

		require.FileExists(t, secretGPGKeyFile)
		content, err = common.ReadFileToString(secretGPGKeyFile)
		assert.NoErrorf(t, err, "File Read Error %s", err)
		assert.Contains(t, content, "PGP PRIVATE KEY BLOCK")
	})
	if err != nil {
		t.Fatal("GPG keys not created as expected")
	}
	t.Run("GPG Read Public Key", func(t *testing.T) {
		key, err = common.ReadFileToString(publicGPGKeyFile)
		entityList, err = GPGReadAmoredKeyRing(key)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		assert.NotNil(t, entityList, "should not be nil")
	})

	t.Run("GPGUnlockKey", func(t *testing.T) {
		key, err = common.ReadFileToString(secretGPGKeyFile)
		entityList, err = GPGReadAmoredKeyRing(key)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		assert.NotNil(t, entityList, "should not be nil")
		if entityList == nil {
			t.Fatal("entityList should not be nil")
		}
		entity, err = GPGSelectEntity(entityList, gpgid)
		assert.NoErrorf(t, err, "select should be no error, but got %v", err)
		assert.NotNil(t, entity, "entity should not be nil")
		err = GPGUnlockKey(entity, keypass)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		if entity != nil {
			assert.False(t, entity.PrivateKey.Encrypted, "encrypted flag should be false")
		}
	})
	plaintextfile := path.Join(test.TestData, "test.gpg.txt")
	//nolint gosec
	err = os.WriteFile(plaintextfile, []byte(plain), 0644)
	require.NoErrorf(t, err, "Create testdata failed")
	cryptedfile := path.Join(test.TestData, "test.gpg.crypt")
	t.Run("Encrypt GPG File", func(t *testing.T) {
		err = GPGEncryptFile(plaintextfile, cryptedfile, publicGPGKeyFile)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
	})
	t.Run("Decrypt GPG File", func(t *testing.T) {
		actual := ""
		actual, err = GPGDecryptFile(cryptedfile, secretGPGKeyFile, keypass, "")
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		assert.Equal(t, plain, actual, "should be equal")
	})
}

func TestGopassSecrets(t *testing.T) {
	var err error
	var gpgid string
	var keyPass string

	test.Testinit(t)
	secretGPGKeyFile := path.Join(test.TestDir, "gpg", "test.gpg.key")
	// publicGPGKeyFile := path.Join(test.TestDir, "gpg", "test.asc")
	gpgKeyPassFile := path.Join(test.TestDir, "gpg", "test.gpgpw")
	storeRoot := path.Join(test.TestDir, "pwlib-store")
	keyIDFile := path.Join(storeRoot, ".gpg-id")
	gpgid, err = common.ReadFileToString(keyIDFile)
	require.NoErrorf(t, err, "GetKeyId should be no error, but got %v", err)
	keyPass, err = common.ReadFileToString(gpgKeyPassFile)
	require.NoErrorf(t, err, "GetKeyPass should be no error, but got %v", err)
	t.Run("Check GoPass Root OK", func(t *testing.T) {
		actual, err := checkStoreRoot(storeRoot)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		assert.Equal(t, gpgid, actual, "KeyID not match")
	})
	t.Run("Check Gopass Root Err", func(t *testing.T) {
		actual, err := checkStoreRoot(test.TestDir)
		assert.Error(t, err, "should be error")
		assert.Empty(t, actual, "should be empty")
	})
	t.Run("Find GoPass GPG Files", func(t *testing.T) {
		// store name is not part of result
		sr := filepath.ToSlash(filepath.Dir(storeRoot))
		actual := findGPGFiles(storeRoot)
		expected := []string{"pwlib-store/test/test1.gpg", "pwlib-store/test/test2.gpg", "pwlib-store/passphrase.gpg"}
		assert.Equal(t, len(expected), len(actual), "len should be %d", len(expected))
		t.Log(actual)
		for _, e := range expected {
			n := path.Join(sr, e)
			found := slices.Contains(actual, n)
			assert.True(t, found, "%s not found in result", n)
		}
	})

	t.Run("List Gopass Secrets", func(t *testing.T) {
		actual := ""
		actual, err = GetGopassSecrets(storeRoot, secretGPGKeyFile, keyPass)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(actual, "\n")
		expected := []string{"pwlib-store:passphrase:", "pwlib-store/test:test1:123456", "pwlib-store/test:test2:"}
		assert.Equal(t, len(expected), len(lines), "len should be %d", len(expected))
		t.Log(lines)
		for _, e := range expected {
			found := false
			for _, l := range lines {
				if strings.Contains(l, e) {
					found = true
					break
				}
			}
			assert.True(t, found, "%s not found in result", e)
		}
	})
	t.Run("Decrypt Gopass GPG File", func(t *testing.T) {
		actual := ""
		filename := path.Join(storeRoot, "test", "test1.gpg")
		actual, err = GPGDecryptFile(filename, secretGPGKeyFile, keyPass, "")
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		assert.Equal(t, "123456\n", actual, "should be equal")
	})
	t.Run("GoPass GetPassword", func(t *testing.T) {
		app := "test"
		pass := ""
		pc := NewConfig(app, storeRoot, path.Dir(secretGPGKeyFile), keyPass, typeGopass)
		pc.PrivateKeyFile = secretGPGKeyFile
		pass, err = pc.GetPassword("pwlib-store/test", "test1")
		expected := "123456"
		assert.NoErrorf(t, err, "Got unexpected error: %s", err)
		assert.Equal(t, expected, pass, "Answer not expected. exp:%s,act:%s", expected, pass)
	})
}
