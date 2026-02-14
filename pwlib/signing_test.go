package pwlib

import (
	"os"
	"testing"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSigning(t *testing.T) {
	test.InitTestDirs()
	err := os.Chdir(test.TestDir)
	require.NoError(t, err, "ChDir failed")

	t.Run("Sign and Verify File - GO method - RSA", func(t *testing.T) {
		app := "test_sign_go_rsa"
		testdata := test.TestData
		pc := NewConfig(app, testdata, testdata, app, typeGO)
		_, _, err = GenRsaKey(pc.PubKeyFile, pc.PrivateKeyFile, pc.KeyPass)
		require.NoError(t, err)

		content := "this is some test content to sign"
		err = common.WriteStringToFile(pc.PlainTextFile, content)
		require.NoError(t, err)

		err = pc.SignFile()
		assert.NoError(t, err)
		assert.FileExists(t, pc.SignatureFile)

		valid, err := pc.VerifyFile()
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Sign and Verify File - GO method - ECDSA", func(t *testing.T) {
		app := "test_sign_go_ecdsa"
		testdata := test.TestData
		pc := NewConfig(app, testdata, testdata, app, typeGO)

		// ECDSA needs specific files for NewConfig to maybe set KeyType, but SignFile uses pc.Method
		// and pc.Method = typeGO uses SignFileSSL, which uses SignString, which uses GetKeyTypeFromFile.
		// So we just need to generate ECDSA keys.
		_, _, err := GenEcdsaKey(pc.PubKeyFile, pc.PrivateKeyFile, pc.KeyPass)
		require.NoError(t, err)

		content := "this is some ecdsa test content to sign"
		err = common.WriteStringToFile(pc.PlainTextFile, content)
		require.NoError(t, err)

		err = pc.SignFile()
		assert.NoError(t, err)
		assert.FileExists(t, pc.SignatureFile)

		valid, err := pc.VerifyFile()
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Sign and Verify File - OPENSSL method", func(t *testing.T) {
		app := "test_sign_openssl"
		testdata := test.TestData
		pc := NewConfig(app, testdata, testdata, app, typeOpenssl)

		_, _, err := GenRsaKey(pc.PubKeyFile, pc.PrivateKeyFile, pc.KeyPass)
		require.NoError(t, err)

		content := "this is some openssl test content to sign"
		err = common.WriteStringToFile(pc.PlainTextFile, content)
		require.NoError(t, err)

		err = pc.SignFile()
		assert.NoError(t, err)
		assert.FileExists(t, pc.SignatureFile)

		valid, err := pc.VerifyFile()
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Sign and Verify File - GPG method", func(t *testing.T) {
		app := "test_sign_gpg"
		testdata := test.TestData
		pc := NewConfig(app, testdata, testdata, app, typeGPG)

		// Create GPG keys
		entity, _, err := CreateGPGEntity("Test Sign", "signing test", "test@example.com", pc.KeyPass)
		require.NoError(t, err)
		err = ExportGPGKeyPair(entity, pc.PubKeyFile, pc.PrivateKeyFile)
		require.NoError(t, err)

		content := "this is some gpg test content to sign"
		err = common.WriteStringToFile(pc.PlainTextFile, content)
		require.NoError(t, err)

		err = pc.SignFile()
		assert.NoError(t, err)
		assert.FileExists(t, pc.SignatureFile)

		valid, err := pc.VerifyFile()
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Sign File - unsupported method age", func(t *testing.T) {
		app := "test_sign_age"
		testdata := test.TestData
		pc := NewConfig(app, testdata, testdata, app, typeAge)

		err = pc.SignFile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signing not supported for age")
	})
}
