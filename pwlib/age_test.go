package pwlib

import (
	"os"
	"path"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"
)

const (
	ageapp      = "testage"
	testAgePub  = ageapp + pubAgeExt
	testAgePriv = ageapp + privAgeExt
)

const plainAge = "This is an age test message for encryption"

func TestAge(t *testing.T) {
	var err error
	var identity *age.X25519Identity
	var recipient string
	test.InitTestDirs()

	secretAgeKeyFile := path.Join(test.TestData, testAgePriv)
	publicAgeKeyFile := path.Join(test.TestData, testAgePub)
	_ = os.Remove(publicAgeKeyFile)
	_ = os.Remove(secretAgeKeyFile)

	t.Run("Age Gen Key", func(t *testing.T) {
		identity, recipient, err = CreateAgeIdentity()
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		assert.NotNil(t, identity, "identity should not be nil")
		if identity == nil {
			t.Fatal("identity should not be nil")
		}
		assert.NotEmpty(t, recipient, "recipient should not be empty")

		err = ExportAgeKeyPair(identity, publicAgeKeyFile, secretAgeKeyFile)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		if err != nil {
			t.Fatal("Age keys not created as expected")
		}

		require.FileExists(t, publicAgeKeyFile)
		content := ""
		content, err = common.ReadFileToString(publicAgeKeyFile)
		assert.NoErrorf(t, err, "File Read Error %s", err)
		assert.Contains(t, content, "age1")

		require.FileExists(t, secretAgeKeyFile)
		content, err = common.ReadFileToString(secretAgeKeyFile)
		assert.NoErrorf(t, err, "File Read Error %s", err)
		assert.Contains(t, content, "AGE-SECRET-KEY-")
	})

	plaintextfile := path.Join(test.TestData, "test.age.txt")
	err = common.WriteStringToFile(plaintextfile, plainAge)
	require.NoErrorf(t, err, "Create testdata failed")
	cryptedfile := path.Join(test.TestData, "test.age.crypt")

	t.Run("Encrypt Age File", func(t *testing.T) {
		err = AgeEncryptFile(plaintextfile, cryptedfile, publicAgeKeyFile)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		require.FileExists(t, cryptedfile)
	})

	t.Run("Decrypt Age File", func(t *testing.T) {
		actual := ""
		actual, err = AgeDecryptFile(cryptedfile, secretAgeKeyFile)
		assert.NoErrorf(t, err, "should be no error, but got %v", err)
		assert.Equal(t, plainAge, actual, "should be equal")
	})
}
