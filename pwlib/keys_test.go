package pwlib

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/test"
)

func TestGetKeyTypeFromFile(t *testing.T) {
	test.InitTestDirs()
	err := os.Chdir(test.TestDir)
	require.NoError(t, err, "ChDir failed")

	rsaPubFile := path.Join(test.TestData, "test_key_type_rsa.pub")
	rsaPrivFile := path.Join(test.TestData, "test_key_type_rsa.pem")
	ecdsaPubFile := path.Join(test.TestData, "test_key_type_ecdsa.pub")
	ecdsaPrivFile := path.Join(test.TestData, "test_key_type_ecdsa.pem")
	gpgPubFile := path.Join(test.TestData, "test_key_type_gpg.pub")
	gpgPrivFile := path.Join(test.TestData, "test_key_type_gpg.pem")
	agePubFile := path.Join(test.TestData, "test_key_type_age.pub")
	agePrivFile := path.Join(test.TestData, "test_key_type_age.pem")

	// Generate RSA keys
	_, _, err = GenRsaKey(rsaPubFile, rsaPrivFile, "")
	require.NoError(t, err)

	// Generate ECDSA keys
	_, _, err = GenEcdsaKey(ecdsaPubFile, ecdsaPrivFile, "")
	require.NoError(t, err)

	// Generate GPG keys
	gpgEntity, _, err := CreateGPGEntity("Test User", "Test", "test@example.com", "pass")
	require.NoError(t, err)
	err = ExportGPGKeyPair(gpgEntity, gpgPubFile, gpgPrivFile)
	require.NoError(t, err)

	// Generate Age keys
	ageIdentity, _, err := CreateAgeIdentity()
	require.NoError(t, err)
	err = ExportAgeKeyPair(ageIdentity, agePubFile, agePrivFile)
	require.NoError(t, err)

	t.Run("Detect RSA Public Key", func(t *testing.T) {
		keyType, err := GetKeyTypeFromFile(rsaPubFile)
		assert.NoError(t, err)
		assert.Equal(t, KeyTypeRSA, keyType)
	})

	t.Run("Detect RSA Private Key", func(t *testing.T) {
		keyType, err := GetKeyTypeFromFile(rsaPrivFile)
		assert.NoError(t, err)
		assert.Equal(t, KeyTypeRSA, keyType)
	})

	t.Run("Detect ECDSA Public Key", func(t *testing.T) {
		keyType, err := GetKeyTypeFromFile(ecdsaPubFile)
		assert.NoError(t, err)
		assert.Equal(t, KeyTypeECDSA, keyType)
	})

	t.Run("Detect ECDSA Private Key", func(t *testing.T) {
		keyType, err := GetKeyTypeFromFile(ecdsaPrivFile)
		assert.NoError(t, err)
		assert.Equal(t, KeyTypeECDSA, keyType)
	})

	t.Run("Detect GPG Public Key", func(t *testing.T) {
		keyType, err := GetKeyTypeFromFile(gpgPubFile)
		assert.NoError(t, err)
		assert.Equal(t, KeyTypeGPG, keyType)
	})

	t.Run("Detect GPG Private Key", func(t *testing.T) {
		keyType, err := GetKeyTypeFromFile(gpgPrivFile)
		assert.NoError(t, err)
		assert.Equal(t, KeyTypeGPG, keyType)
	})

	t.Run("Detect Age Public Key", func(t *testing.T) {
		keyType, err := GetKeyTypeFromFile(agePubFile)
		assert.NoError(t, err)
		assert.Equal(t, KeyTypeAGE, keyType)
	})

	t.Run("Detect Age Private Key", func(t *testing.T) {
		keyType, err := GetKeyTypeFromFile(agePrivFile)
		assert.NoError(t, err)
		assert.Equal(t, KeyTypeAGE, keyType)
	})

	t.Run("Non-existent file", func(t *testing.T) {
		keyType, err := GetKeyTypeFromFile("non_existent.pem")
		assert.Error(t, err)
		assert.Equal(t, KeyTypeUnknown, keyType)
	})

	t.Run("Invalid PEM file", func(t *testing.T) {
		invalidFile := path.Join(test.TestData, "invalid.pem")
		err = os.WriteFile(invalidFile, []byte("not a pem"), 0600)
		require.NoError(t, err)

		keyType, err := GetKeyTypeFromFile(invalidFile)
		assert.Error(t, err)
		assert.Equal(t, KeyTypeUnknown, keyType)
	})
}
