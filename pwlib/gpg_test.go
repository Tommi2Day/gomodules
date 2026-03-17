package pwlib

import (
	"fmt"
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

func TestGPGDetectRecipients(t *testing.T) {
	test.InitTestDirs()

	// re-use key files created by TestGPG when run in the same suite,
	// or create fresh ones here
	pubKeyFile := path.Join(test.TestData, testGPGpub)
	privKeyFile := path.Join(test.TestData, testGPGPriv)
	_ = os.Remove(pubKeyFile)
	_ = os.Remove(privKeyFile)
	entity, gpgid, err := CreateGPGEntity(testGPGName, "DetectTest", testGPGEmail, testGPGPass)
	require.NoErrorf(t, err, "GPG key creation failed: %v", err)
	err = ExportGPGKeyPair(entity, pubKeyFile, privKeyFile)
	require.NoErrorf(t, err, "GPG key export failed: %v", err)

	plaintextFile := path.Join(test.TestData, "detect_gpg.txt")
	cryptedFile := path.Join(test.TestData, "detect_gpg.gpg")
	err = common.WriteStringToFile(plaintextFile, "detect-recipient-test")
	require.NoErrorf(t, err, "write plaintext failed: %v", err)
	err = GPGEncryptFile(plaintextFile, cryptedFile, pubKeyFile)
	require.NoErrorf(t, err, "encrypt failed: %v", err)

	t.Run("detects at least one recipient key ID", func(t *testing.T) {
		keyIDs, dErr := GPGDetectRecipients(cryptedFile)
		assert.NoError(t, dErr)
		assert.NotEmpty(t, keyIDs, "should find at least one recipient key ID")
		t.Logf("detected recipient key IDs: %v", keyIDs)
	})

	t.Run("detected key ID belongs to the encryption key or subkey", func(t *testing.T) {
		keyIDs, dErr := GPGDetectRecipients(cryptedFile)
		require.NoError(t, dErr)
		// build the set of all known key IDs for this entity
		known := map[string]bool{
			fmt.Sprintf("%016X", entity.PrimaryKey.KeyId): true,
		}
		for _, sk := range entity.Subkeys {
			known[fmt.Sprintf("%016X", sk.PublicKey.KeyId)] = true
		}
		matched := false
		for _, kid := range keyIDs {
			if known[kid] {
				matched = true
			}
		}
		assert.Truef(t, matched, "none of the recipient IDs %v match the entity key ID %s or its subkeys", keyIDs, gpgid)
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, dErr := GPGDetectRecipients(path.Join(test.TestData, "no-such-file.gpg"))
		assert.Error(t, dErr)
	})
}

func TestGPGFindDecryptKey(t *testing.T) {
	test.InitTestDirs()

	// create two independent key pairs
	pubA := path.Join(test.TestData, "find_key_a"+pubGPGExt)
	privA := path.Join(test.TestData, "find_key_a"+privGPGExt)
	pubB := path.Join(test.TestData, "find_key_b"+pubGPGExt)
	privB := path.Join(test.TestData, "find_key_b"+privGPGExt)
	for _, f := range []string{pubA, privA, pubB, privB} {
		_ = os.Remove(f)
	}

	entityA, _, err := CreateGPGEntity("User A", "FindKeyTest", "a@example.com", testGPGPass)
	require.NoError(t, err)
	err = ExportGPGKeyPair(entityA, pubA, privA)
	require.NoError(t, err)

	entityB, _, err := CreateGPGEntity("User B", "FindKeyTest", "b@example.com", testGPGPass)
	require.NoError(t, err)
	err = ExportGPGKeyPair(entityB, pubB, privB)
	require.NoError(t, err)

	plaintextFile := path.Join(test.TestData, "find_key.txt")
	cryptedFile := path.Join(test.TestData, "find_key.gpg")
	err = common.WriteStringToFile(plaintextFile, "key-detection-test")
	require.NoError(t, err)
	// encrypt only to key A
	err = GPGEncryptFile(plaintextFile, cryptedFile, pubA)
	require.NoError(t, err)

	t.Run("finds matching key among multiple candidates", func(t *testing.T) {
		matched, findErr := GPGFindDecryptKey(cryptedFile, []string{privB, privA})
		assert.NoError(t, findErr)
		assert.Equal(t, privA, matched, "should match key A, not key B")
	})

	t.Run("finds key when it is the only candidate", func(t *testing.T) {
		matched, findErr := GPGFindDecryptKey(cryptedFile, []string{privA})
		assert.NoError(t, findErr)
		assert.Equal(t, privA, matched)
	})

	t.Run("returns error when no key matches", func(t *testing.T) {
		matched, findErr := GPGFindDecryptKey(cryptedFile, []string{privB})
		assert.Error(t, findErr)
		assert.Empty(t, matched)
	})

	t.Run("returns error for empty key list", func(t *testing.T) {
		matched, findErr := GPGFindDecryptKey(cryptedFile, []string{})
		assert.Error(t, findErr)
		assert.Empty(t, matched)
	})

	t.Run("matched key can actually decrypt the file", func(t *testing.T) {
		matched, findErr := GPGFindDecryptKey(cryptedFile, []string{privB, privA})
		require.NoError(t, findErr)
		content, decErr := GPGDecryptFile(cryptedFile, matched, testGPGPass, "")
		assert.NoError(t, decErr)
		assert.Equal(t, "key-detection-test", content)
	})

	t.Run("nonexistent encrypted file returns error", func(t *testing.T) {
		_, findErr := GPGFindDecryptKey(path.Join(test.TestData, "no-such.gpg"), []string{privA})
		assert.Error(t, findErr)
	})
}

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

func TestGPGMultiRecipient(t *testing.T) {
	test.InitTestDirs()

	// create three independent GPG key pairs
	type gpgPair struct{ pub, priv string }
	var pairs [3]gpgPair
	var entities [3]*openpgp.Entity
	for i := range pairs {
		pairs[i] = gpgPair{
			pub:  path.Join(test.TestData, fmt.Sprintf("multi_gpg_%d"+pubGPGExt, i)),
			priv: path.Join(test.TestData, fmt.Sprintf("multi_gpg_%d"+privGPGExt, i)),
		}
		entity, _, err := CreateGPGEntity(testGPGName, fmt.Sprintf("Multi%d", i), testGPGEmail, testGPGPass)
		require.NoError(t, err)
		require.NoError(t, ExportGPGKeyPair(entity, pairs[i].pub, pairs[i].priv))
		entities[i] = entity
	}

	plaintextFile := path.Join(test.TestData, "multi_gpg.txt")
	cryptedFile := path.Join(test.TestData, "multi_gpg.gpg")
	require.NoError(t, common.WriteStringToFile(plaintextFile, plain))

	t.Run("encrypt to multiple public key files", func(t *testing.T) {
		err := GPGEncryptFileMulti(plaintextFile, cryptedFile, []string{pairs[0].pub, pairs[1].pub, pairs[2].pub})
		assert.NoError(t, err)
		assert.FileExists(t, cryptedFile)
	})

	t.Run("each recipient can decrypt independently", func(t *testing.T) {
		for i, p := range pairs {
			content, err := GPGDecryptFile(cryptedFile, p.priv, testGPGPass, "")
			assert.NoErrorf(t, err, "key %d should decrypt", i)
			assert.Equal(t, plain, content)
		}
	})

	t.Run("GPGDecryptFileMulti finds first matching key", func(t *testing.T) {
		content, err := GPGDecryptFileMulti(cryptedFile, []string{pairs[0].priv, pairs[1].priv, pairs[2].priv}, testGPGPass)
		assert.NoError(t, err)
		assert.Equal(t, plain, content)
	})

	t.Run("GPGDecryptFileMulti works regardless of candidate order", func(t *testing.T) {
		content, err := GPGDecryptFileMulti(cryptedFile, []string{pairs[2].priv, pairs[1].priv, pairs[0].priv}, testGPGPass)
		assert.NoError(t, err)
		assert.Equal(t, plain, content)
	})

	t.Run("GPGDecryptFileMulti returns error when no key matches", func(t *testing.T) {
		otherEntity, _, err := CreateGPGEntity(testGPGName, "Other", testGPGEmail, testGPGPass)
		require.NoError(t, err)
		otherPriv := path.Join(test.TestData, "multi_gpg_other"+privGPGExt)
		otherPub := path.Join(test.TestData, "multi_gpg_other"+pubGPGExt)
		require.NoError(t, ExportGPGKeyPair(otherEntity, otherPub, otherPriv))
		_, err = GPGDecryptFileMulti(cryptedFile, []string{otherPriv}, testGPGPass)
		assert.Error(t, err)
	})

	t.Run("GPGEncryptFileMulti with empty key list returns error", func(t *testing.T) {
		err := GPGEncryptFileMulti(plaintextFile, cryptedFile, []string{})
		assert.Error(t, err)
	})

	t.Run("GPGEncryptFileMulti with nonexistent key file returns error", func(t *testing.T) {
		err := GPGEncryptFileMulti(plaintextFile, cryptedFile, []string{"/no/such/key.pub"})
		assert.Error(t, err)
	})

	t.Run("GPGEncryptFileMulti with nonexistent plain file returns error", func(t *testing.T) {
		err := GPGEncryptFileMulti(path.Join(test.TestData, "no-such.txt"), cryptedFile, []string{pairs[0].pub})
		assert.Error(t, err)
	})
}
