package pwlib

import (
	"os"
	"path"
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

// const testGPGPFILE = "test.gpgpw"
const gpgapp = "testgpg"
const testGPGpub = gpgapp + pubGPGExt
const testGPGPriv = gpgapp + privGPGExt

func TestGPG(t *testing.T) {
	var err error
	var gpgid string
	var keypass string
	var key string
	var entityList openpgp.EntityList
	var entity *openpgp.Entity
	test.InitTestDirs()

	secretGPGKeyFile := path.Join(test.TestData, testGPGPriv)
	publicGPGKeyFile := path.Join(test.TestData, testGPGpub)
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
	err = common.WriteStringToFile(plaintextfile, plain)
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
