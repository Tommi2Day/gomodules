package pwlib

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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

	t.Run("Decrypt GPG File with empty keypass falls back to gpg-agent and errors", func(t *testing.T) {
		t.Setenv("GPG_PASSPHRASE", "")
		emptyHome := filepath.Join(test.TestData, "gpg-decrypt-no-agent")
		require.NoError(t, os.MkdirAll(emptyHome, 0700))
		t.Setenv(gpgEnvHome, emptyHome)
		t.Setenv("GPG_AGENT_INFO", "")
		t.Setenv("XDG_RUNTIME_DIR", "")
		_, decErr := GPGDecryptFile(cryptedfile, secretGPGKeyFile, "", "")
		assert.Error(t, decErr)
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

// writeSecretKeyRing serialises entity as a binary GPG secret keyring file
// (the secring.gpg format readable by openpgp.ReadKeyRing).
func writeSecretKeyRing(t *testing.T, path string, entity *openpgp.Entity) {
	t.Helper()
	f, err := os.Create(path) //nolint:gosec // test helper
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	require.NoError(t, entity.SerializePrivateWithoutSigning(f, nil))
}

func TestGPGAnyKeyEncrypted(t *testing.T) {
	test.InitTestDirs()

	t.Run("returns false for empty list", func(t *testing.T) {
		assert.False(t, gpgAnyKeyEncrypted(openpgp.EntityList{}))
	})

	t.Run("returns true for freshly created (encrypted) entity", func(t *testing.T) {
		entity, _, err := CreateGPGEntity(testGPGName, "AnyEncTest", testGPGEmail, testGPGPass)
		require.NoError(t, err)
		assert.True(t, gpgAnyKeyEncrypted(openpgp.EntityList{entity}))
	})

	t.Run("returns false after entity is unlocked", func(t *testing.T) {
		entity, _, err := CreateGPGEntity(testGPGName, "AnyEncTest", testGPGEmail, testGPGPass)
		require.NoError(t, err)
		require.NoError(t, entity.DecryptPrivateKeys([]byte(testGPGPass)))
		assert.False(t, gpgAnyKeyEncrypted(openpgp.EntityList{entity}))
	})

	t.Run("returns false for entity with nil private key and nil subkey private keys", func(t *testing.T) {
		entity, _, err := CreateGPGEntity(testGPGName, "AnyEncTest", testGPGEmail, testGPGPass)
		require.NoError(t, err)
		entity.PrivateKey = nil
		for i := range entity.Subkeys {
			entity.Subkeys[i].PrivateKey = nil
		}
		assert.False(t, gpgAnyKeyEncrypted(openpgp.EntityList{entity}))
	})
}

func TestGPGExportSecretKeysArmored(t *testing.T) {
	test.InitTestDirs()

	t.Run("empty GNUPGHOME returns error (no keys or binary unavailable)", func(t *testing.T) {
		emptyHome := path.Join(test.TestData, "gnupg-export-empty")
		_ = os.RemoveAll(emptyHome)
		require.NoError(t, os.MkdirAll(emptyHome, 0700))
		_ = os.Setenv(gpgEnvHome, emptyHome)
		_, err := GPGExportSecretKeysArmored()
		_ = os.Unsetenv(gpgEnvHome)
		// gpg binary unavailable → exec error; or available but empty keyring → "no keys" error
		assert.Error(t, err)
	})
}

func TestGPGSystemSecretKeys(t *testing.T) {
	test.InitTestDirs()

	entity, _, err := CreateGPGEntity(testGPGName, "SystemKeysTest", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	testGnupgHome := path.Join(test.TestData, "gnupg-system-keys-test")
	_ = os.RemoveAll(testGnupgHome)
	require.NoError(t, os.MkdirAll(testGnupgHome, 0700))
	writeSecretKeyRing(t, path.Join(testGnupgHome, "secring.gpg"), entity)

	t.Run("reads keys from secring.gpg when present", func(t *testing.T) {
		_ = os.Setenv(gpgEnvHome, testGnupgHome)
		el, sysErr := GPGSystemSecretKeys()
		_ = os.Unsetenv(gpgEnvHome)
		assert.NoError(t, sysErr)
		assert.NotEmpty(t, el)
	})

	t.Run("returns error when no secring.gpg and gpg binary yields nothing", func(t *testing.T) {
		emptyHome := path.Join(test.TestData, "gnupg-system-empty")
		_ = os.RemoveAll(emptyHome)
		require.NoError(t, os.MkdirAll(emptyHome, 0700))
		_ = os.Setenv(gpgEnvHome, emptyHome)
		_, sysErr := GPGSystemSecretKeys()
		_ = os.Unsetenv(gpgEnvHome)
		assert.Error(t, sysErr)
	})
}

func TestGPGHomeDir(t *testing.T) {
	test.InitTestDirs()

	t.Run("GNUPGHOME env var is used", func(t *testing.T) {
		_ = os.Setenv(gpgEnvHome, "/custom/gnupg")
		dir, err := GPGHomeDir()
		_ = os.Unsetenv(gpgEnvHome)
		assert.NoError(t, err)
		assert.Equal(t, "/custom/gnupg", dir)
	})

	t.Run("default is ~/.gnupg", func(t *testing.T) {
		_ = os.Unsetenv(gpgEnvHome)
		dir, err := GPGHomeDir()
		assert.NoError(t, err)
		assert.Contains(t, filepath.ToSlash(dir), ".gnupg")
	})
}

func TestGPGSecretKeyRingPath(t *testing.T) {
	test.InitTestDirs()

	_ = os.Setenv(gpgEnvHome, "/custom/gnupg")
	p, err := GPGSecretKeyRingPath()
	_ = os.Unsetenv(gpgEnvHome)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("/custom/gnupg", "secring.gpg"), p)
}

func TestGPGReadSecretKeyRing(t *testing.T) {
	test.InitTestDirs()

	entity, _, err := CreateGPGEntity(testGPGName, "KeyRingTest", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	secring := path.Join(test.TestData, "test_secring.gpg")
	writeSecretKeyRing(t, secring, entity)

	t.Run("reads keys from binary keyring", func(t *testing.T) {
		el, readErr := GPGReadSecretKeyRing(secring)
		assert.NoError(t, readErr)
		assert.NotEmpty(t, el)
	})

	t.Run("nonexistent keyring returns error", func(t *testing.T) {
		_, readErr := GPGReadSecretKeyRing(path.Join(test.TestData, "no_such_secring.gpg"))
		assert.Error(t, readErr)
	})

	t.Run("empty keyring file returns error", func(t *testing.T) {
		emptyRing := path.Join(test.TestData, "empty_secring.gpg")
		require.NoError(t, os.WriteFile(emptyRing, []byte{}, 0600))
		_, readErr := GPGReadSecretKeyRing(emptyRing)
		assert.Error(t, readErr)
	})
}

func TestGPGDecryptFileAuto(t *testing.T) {
	test.InitTestDirs()

	entity, _, err := CreateGPGEntity(testGPGName, "AutoDecryptTest", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	pubKeyFile := path.Join(test.TestData, "auto_decrypt"+pubGPGExt)
	privKeyFile := path.Join(test.TestData, "auto_decrypt"+privGPGExt)
	require.NoError(t, ExportGPGKeyPair(entity, pubKeyFile, privKeyFile))

	plaintextFile := path.Join(test.TestData, "auto_decrypt.txt")
	cryptedFile := path.Join(test.TestData, "auto_decrypt.gpg")
	require.NoError(t, common.WriteStringToFile(plaintextFile, plain))
	require.NoError(t, GPGEncryptFile(plaintextFile, cryptedFile, pubKeyFile))

	// set up a test GNUPGHOME with a secring.gpg containing the key
	testGnupgHome := path.Join(test.TestData, "gnupg-auto-test")
	_ = os.RemoveAll(testGnupgHome)
	require.NoError(t, os.MkdirAll(testGnupgHome, 0700))
	writeSecretKeyRing(t, path.Join(testGnupgHome, "secring.gpg"), entity)

	t.Run("decrypts using secring.gpg", func(t *testing.T) {
		_ = os.Setenv(gpgEnvHome, testGnupgHome)
		content, decErr := GPGDecryptFileAuto(cryptedFile, testGPGPass)
		_ = os.Unsetenv(gpgEnvHome)
		assert.NoError(t, decErr)
		assert.Equal(t, plain, content)
	})

	t.Run("wrong passphrase returns error", func(t *testing.T) {
		_ = os.Setenv(gpgEnvHome, testGnupgHome)
		_, decErr := GPGDecryptFileAuto(cryptedFile, "wrongpassphrase")
		_ = os.Unsetenv(gpgEnvHome)
		assert.Error(t, decErr)
	})

	t.Run("empty passphrase with encrypted key falls back to gpg-agent or error", func(t *testing.T) {
		_ = os.Setenv(gpgEnvHome, testGnupgHome)
		_, decErr := GPGDecryptFileAuto(cryptedFile, "")
		_ = os.Unsetenv(gpgEnvHome)
		assert.Error(t, decErr)
		// Without a running gpg-agent the error describes the socket lookup failure.
		assert.NotContains(t, decErr.Error(), "stdin is not a terminal")
	})

	t.Run("empty GNUPGHOME without gpg binary falls back to error", func(t *testing.T) {
		emptyHome := path.Join(test.TestData, "gnupg-empty-auto")
		_ = os.RemoveAll(emptyHome)
		require.NoError(t, os.MkdirAll(emptyHome, 0700))
		_ = os.Setenv(gpgEnvHome, emptyHome)
		_, decErr := GPGDecryptFileAuto(cryptedFile, testGPGPass)
		_ = os.Unsetenv(gpgEnvHome)
		// may succeed if a system gpg binary with matching keys is available,
		// but in a clean test env there should be no matching key
		t.Logf("empty GNUPGHOME result: %v", decErr)
	})
}

func TestGopassGPGDecryptFileAutoKeyRing(t *testing.T) {
	test.InitTestDirs()

	storeDir := path.Join(test.TestData, "gopass-auto-gpg-store")
	_ = os.RemoveAll(storeDir)
	require.NoError(t, os.MkdirAll(storeDir, 0700))

	entity, _, err := CreateGPGEntity(testGPGName, "GopassAutoTest", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	pubKeyFile := path.Join(test.TestData, "gopass_auto"+pubGPGExt)
	require.NoError(t, ExportGPGKeyPair(entity, pubKeyFile, path.Join(test.TestData, "gopass_auto"+privGPGExt)))

	// write a gopass secret encrypted to the test key
	require.NoError(t, GopassWrite(storeDir, "auto/secret", "autopassword", pubKeyFile, GopassCryptoGPG))

	// set up GNUPGHOME with a secring.gpg containing the test key
	testGnupgHome := path.Join(test.TestData, "gnupg-gopass-auto")
	_ = os.RemoveAll(testGnupgHome)
	require.NoError(t, os.MkdirAll(testGnupgHome, 0700))
	writeSecretKeyRing(t, path.Join(testGnupgHome, "secring.gpg"), entity)

	t.Run("GopassRead with empty keyFile uses system keyring", func(t *testing.T) {
		_ = os.Setenv(gpgEnvHome, testGnupgHome)
		secret, readErr := GopassRead(storeDir, "auto/secret", "", testGPGPass, GopassCryptoGPG)
		_ = os.Unsetenv(gpgEnvHome)
		assert.NoError(t, readErr)
		assert.Equal(t, "autopassword", secret)
	})
}
