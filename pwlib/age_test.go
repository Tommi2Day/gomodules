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

func TestAgeDetectIdentity(t *testing.T) {
	test.InitTestDirs()

	// create two independent age key pairs
	pubA := path.Join(test.TestData, "detect_age_a"+pubAgeExt)
	privA := path.Join(test.TestData, "detect_age_a"+privAgeExt)
	pubB := path.Join(test.TestData, "detect_age_b"+pubAgeExt)
	privB := path.Join(test.TestData, "detect_age_b"+privAgeExt)
	for _, f := range []string{pubA, privA, pubB, privB} {
		_ = os.Remove(f)
	}

	identityA, _, err := CreateAgeIdentity()
	require.NoError(t, err)
	err = ExportAgeKeyPair(identityA, pubA, privA)
	require.NoError(t, err)

	identityB, _, err := CreateAgeIdentity()
	require.NoError(t, err)
	err = ExportAgeKeyPair(identityB, pubB, privB)
	require.NoError(t, err)

	plaintextFile := path.Join(test.TestData, "detect_age.txt")
	cryptedFile := path.Join(test.TestData, "detect_age.age")
	err = common.WriteStringToFile(plaintextFile, "age-identity-detection-test")
	require.NoError(t, err)
	// encrypt only to identity A
	err = AgeEncryptFile(plaintextFile, cryptedFile, pubA)
	require.NoError(t, err)

	t.Run("finds matching identity among multiple candidates", func(t *testing.T) {
		matched, dErr := AgeDetectIdentity(cryptedFile, []string{privB, privA})
		assert.NoError(t, dErr)
		assert.Equal(t, privA, matched, "should match identity A, not identity B")
	})

	t.Run("finds identity when it is the only candidate", func(t *testing.T) {
		matched, dErr := AgeDetectIdentity(cryptedFile, []string{privA})
		assert.NoError(t, dErr)
		assert.Equal(t, privA, matched)
	})

	t.Run("returns error when no identity matches", func(t *testing.T) {
		matched, dErr := AgeDetectIdentity(cryptedFile, []string{privB})
		assert.Error(t, dErr)
		assert.Empty(t, matched)
	})

	t.Run("returns error for empty identity list", func(t *testing.T) {
		matched, dErr := AgeDetectIdentity(cryptedFile, []string{})
		assert.Error(t, dErr)
		assert.Empty(t, matched)
	})

	t.Run("matched identity can actually decrypt the file", func(t *testing.T) {
		matched, dErr := AgeDetectIdentity(cryptedFile, []string{privB, privA})
		require.NoError(t, dErr)
		content, decErr := AgeDecryptFile(cryptedFile, matched)
		assert.NoError(t, decErr)
		assert.Equal(t, "age-identity-detection-test", content)
	})

	t.Run("nonexistent encrypted file returns error", func(t *testing.T) {
		_, dErr := AgeDetectIdentity(path.Join(test.TestData, "no-such.age"), []string{privA})
		assert.Error(t, dErr)
	})

	t.Run("nonexistent identity file is skipped gracefully", func(t *testing.T) {
		// privA is valid, so it should still match even with a bad file first
		matched, dErr := AgeDetectIdentity(cryptedFile, []string{"/nonexistent/id.age", privA})
		assert.NoError(t, dErr)
		assert.Equal(t, privA, matched)
	})
}

const testAgePassphrase = "s3cr3tAgeP@ss!"

func TestAgePassphrase(t *testing.T) {
	test.InitTestDirs()

	plaintextFile := path.Join(test.TestData, "age_pass.txt")
	cryptedFile := path.Join(test.TestData, "age_pass.age")
	require.NoError(t, common.WriteStringToFile(plaintextFile, plainAge))

	t.Run("encrypt with passphrase", func(t *testing.T) {
		err := AgeEncryptFileWithPassphrase(plaintextFile, cryptedFile, testAgePassphrase)
		assert.NoError(t, err)
		assert.FileExists(t, cryptedFile)
	})

	t.Run("decrypt with correct passphrase", func(t *testing.T) {
		content, err := AgeDecryptFileWithPassphrase(cryptedFile, testAgePassphrase)
		assert.NoError(t, err)
		assert.Equal(t, plainAge, content)
	})

	t.Run("decrypt with wrong passphrase returns error", func(t *testing.T) {
		_, err := AgeDecryptFileWithPassphrase(cryptedFile, "wrongpassphrase")
		assert.Error(t, err)
	})

	t.Run("decrypt with empty passphrase returns error", func(t *testing.T) {
		_, err := AgeDecryptFileWithPassphrase(cryptedFile, "")
		assert.Error(t, err)
	})

	t.Run("passphrase-encrypted file cannot be decrypted as key-pair file", func(t *testing.T) {
		// generate a key pair and try to use it to decrypt a passphrase file
		id, _, err := CreateAgeIdentity()
		require.NoError(t, err)
		privFile := path.Join(test.TestData, "age_pass_wrong"+privAgeExt)
		pubFile := path.Join(test.TestData, "age_pass_wrong"+pubAgeExt)
		require.NoError(t, ExportAgeKeyPair(id, pubFile, privFile))
		_, err = AgeDecryptFile(cryptedFile, privFile)
		assert.Error(t, err)
	})

	t.Run("encrypt with empty passphrase returns error", func(t *testing.T) {
		err := AgeEncryptFileWithPassphrase(plaintextFile, cryptedFile, "")
		assert.Error(t, err)
	})

	t.Run("nonexistent plain file returns error", func(t *testing.T) {
		err := AgeEncryptFileWithPassphrase(path.Join(test.TestData, "no-such.txt"), cryptedFile, testAgePassphrase)
		assert.Error(t, err)
	})
}

func TestAgeEncryptedIdentity(t *testing.T) {
	test.InitTestDirs()

	pubFile := path.Join(test.TestData, "age_encid"+pubAgeExt)
	encPrivFile := path.Join(test.TestData, "age_encid"+privAgeExt)
	_ = os.Remove(pubFile)
	_ = os.Remove(encPrivFile)

	identity, _, err := CreateAgeIdentity()
	require.NoError(t, err)

	t.Run("ExportAgeKeyPairEncrypted creates files", func(t *testing.T) {
		err = ExportAgeKeyPairEncrypted(identity, pubFile, encPrivFile, testAgePassphrase)
		assert.NoError(t, err)
		assert.FileExists(t, pubFile)
		assert.FileExists(t, encPrivFile)
	})

	t.Run("exported public key is plaintext bech32", func(t *testing.T) {
		content, err := common.ReadFileToString(pubFile)
		assert.NoError(t, err)
		assert.Contains(t, content, "age1")
	})

	t.Run("exported private key is an age-encrypted binary (not plaintext)", func(t *testing.T) {
		content, err := common.ReadFileToString(encPrivFile)
		assert.NoError(t, err)
		assert.NotContains(t, content, "AGE-SECRET-KEY-",
			"encrypted identity file must not contain the plaintext key")
	})

	t.Run("AgeLoadEncryptedIdentity with correct passphrase", func(t *testing.T) {
		loaded, loadErr := AgeLoadEncryptedIdentity(encPrivFile, testAgePassphrase)
		assert.NoError(t, loadErr)
		require.NotNil(t, loaded)
		assert.Equal(t, identity.Recipient().String(), loaded.Recipient().String(),
			"loaded identity must match the original")
	})

	t.Run("AgeLoadEncryptedIdentity with wrong passphrase returns error", func(t *testing.T) {
		_, loadErr := AgeLoadEncryptedIdentity(encPrivFile, "wrongpassphrase")
		assert.Error(t, loadErr)
	})

	t.Run("AgeLoadEncryptedIdentity with nonexistent file returns error", func(t *testing.T) {
		_, loadErr := AgeLoadEncryptedIdentity(path.Join(test.TestData, "no-such.age"), testAgePassphrase)
		assert.Error(t, loadErr)
	})

	// encrypt a data file to the identity, then decrypt using the encrypted private key
	plaintextFile := path.Join(test.TestData, "age_encid_data.txt")
	cryptedFile := path.Join(test.TestData, "age_encid_data.age")
	require.NoError(t, common.WriteStringToFile(plaintextFile, plainAge))
	require.NoError(t, AgeEncryptFile(plaintextFile, cryptedFile, pubFile))

	t.Run("AgeDecryptFileWithEncryptedIdentity correct passphrase", func(t *testing.T) {
		content, decErr := AgeDecryptFileWithEncryptedIdentity(cryptedFile, encPrivFile, testAgePassphrase)
		assert.NoError(t, decErr)
		assert.Equal(t, plainAge, content)
	})

	t.Run("AgeDecryptFileWithEncryptedIdentity wrong passphrase returns error", func(t *testing.T) {
		_, decErr := AgeDecryptFileWithEncryptedIdentity(cryptedFile, encPrivFile, "wrongpassphrase")
		assert.Error(t, decErr)
	})

	t.Run("ExportAgeKeyPairEncrypted nil identity returns error", func(t *testing.T) {
		err = ExportAgeKeyPairEncrypted(nil, pubFile, encPrivFile, testAgePassphrase)
		assert.Error(t, err)
	})
}

func TestAgeDetectIdentityWithPassphrase(t *testing.T) {
	test.InitTestDirs()

	pubA := path.Join(test.TestData, "detect_enc_age_a"+pubAgeExt)
	encPrivA := path.Join(test.TestData, "detect_enc_age_a"+privAgeExt)
	pubB := path.Join(test.TestData, "detect_enc_age_b"+pubAgeExt)
	encPrivB := path.Join(test.TestData, "detect_enc_age_b"+privAgeExt)

	idA, _, err := CreateAgeIdentity()
	require.NoError(t, err)
	require.NoError(t, ExportAgeKeyPairEncrypted(idA, pubA, encPrivA, testAgePassphrase))

	idB, _, err := CreateAgeIdentity()
	require.NoError(t, err)
	require.NoError(t, ExportAgeKeyPairEncrypted(idB, pubB, encPrivB, testAgePassphrase))

	plaintextFile := path.Join(test.TestData, "detect_enc_age.txt")
	cryptedFile := path.Join(test.TestData, "detect_enc_age.age")
	require.NoError(t, common.WriteStringToFile(plaintextFile, "detect-encrypted-identity-test"))
	require.NoError(t, AgeEncryptFile(plaintextFile, cryptedFile, pubA))

	t.Run("finds matching encrypted identity among multiple candidates", func(t *testing.T) {
		matched, dErr := AgeDetectIdentityWithPassphrase(cryptedFile, []string{encPrivB, encPrivA}, testAgePassphrase)
		assert.NoError(t, dErr)
		assert.Equal(t, encPrivA, matched)
	})

	t.Run("returns error when no identity matches", func(t *testing.T) {
		matched, dErr := AgeDetectIdentityWithPassphrase(cryptedFile, []string{encPrivB}, testAgePassphrase)
		assert.Error(t, dErr)
		assert.Empty(t, matched)
	})

	t.Run("wrong passphrase causes all candidates to be skipped, returns error", func(t *testing.T) {
		matched, dErr := AgeDetectIdentityWithPassphrase(cryptedFile, []string{encPrivA, encPrivB}, "wrongpass")
		assert.Error(t, dErr)
		assert.Empty(t, matched)
	})

	t.Run("matched encrypted identity can actually decrypt the file", func(t *testing.T) {
		matched, dErr := AgeDetectIdentityWithPassphrase(cryptedFile, []string{encPrivB, encPrivA}, testAgePassphrase)
		require.NoError(t, dErr)
		content, decErr := AgeDecryptFileWithEncryptedIdentity(cryptedFile, matched, testAgePassphrase)
		assert.NoError(t, decErr)
		assert.Equal(t, "detect-encrypted-identity-test", content)
	})

	t.Run("nonexistent encrypted file returns error", func(t *testing.T) {
		_, dErr := AgeDetectIdentityWithPassphrase(path.Join(test.TestData, "no-such.age"),
			[]string{encPrivA}, testAgePassphrase)
		assert.Error(t, dErr)
	})
}

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

func TestAgeMultiRecipient(t *testing.T) {
	test.InitTestDirs()

	// create three independent age key pairs
	pubA := path.Join(test.TestData, "multi_age_a"+pubAgeExt)
	privA := path.Join(test.TestData, "multi_age_a"+privAgeExt)
	pubB := path.Join(test.TestData, "multi_age_b"+pubAgeExt)
	privB := path.Join(test.TestData, "multi_age_b"+privAgeExt)
	pubC := path.Join(test.TestData, "multi_age_c"+pubAgeExt)
	privC := path.Join(test.TestData, "multi_age_c"+privAgeExt)

	for _, id := range []struct{ pub, priv string }{{pubA, privA}, {pubB, privB}, {pubC, privC}} {
		identity, _, err := CreateAgeIdentity()
		require.NoError(t, err)
		require.NoError(t, ExportAgeKeyPair(identity, id.pub, id.priv))
	}

	plaintextFile := path.Join(test.TestData, "multi_age.txt")
	cryptedFile := path.Join(test.TestData, "multi_age.age")
	require.NoError(t, common.WriteStringToFile(plaintextFile, plainAge))

	t.Run("encrypt to multiple recipient files", func(t *testing.T) {
		err := AgeEncryptFileMulti(plaintextFile, cryptedFile, []string{pubA, pubB, pubC})
		assert.NoError(t, err)
		assert.FileExists(t, cryptedFile)
	})

	t.Run("each recipient can decrypt independently", func(t *testing.T) {
		for name, priv := range map[string]string{"A": privA, "B": privB, "C": privC} {
			content, err := AgeDecryptFile(cryptedFile, priv)
			assert.NoErrorf(t, err, "identity %s should decrypt", name)
			assert.Equal(t, plainAge, content)
		}
	})

	t.Run("AgeDecryptFileMulti finds first matching identity", func(t *testing.T) {
		content, err := AgeDecryptFileMulti(cryptedFile, []string{privA, privB, privC})
		assert.NoError(t, err)
		assert.Equal(t, plainAge, content)
	})

	t.Run("AgeDecryptFileMulti works regardless of candidate order", func(t *testing.T) {
		content, err := AgeDecryptFileMulti(cryptedFile, []string{privC, privB, privA})
		assert.NoError(t, err)
		assert.Equal(t, plainAge, content)
	})

	t.Run("AgeDecryptFileMulti returns error when no identity matches", func(t *testing.T) {
		otherID, _, err := CreateAgeIdentity()
		require.NoError(t, err)
		otherPriv := path.Join(test.TestData, "multi_age_other"+privAgeExt)
		otherPub := path.Join(test.TestData, "multi_age_other"+pubAgeExt)
		require.NoError(t, ExportAgeKeyPair(otherID, otherPub, otherPriv))
		_, err = AgeDecryptFileMulti(cryptedFile, []string{otherPriv})
		assert.Error(t, err)
	})

	t.Run("AgeEncryptFileMulti with empty recipient list returns error", func(t *testing.T) {
		err := AgeEncryptFileMulti(plaintextFile, cryptedFile, []string{})
		assert.Error(t, err)
	})

	t.Run("AgeEncryptFileMulti with nonexistent recipient file returns error", func(t *testing.T) {
		err := AgeEncryptFileMulti(plaintextFile, cryptedFile, []string{"/no/such/key.pub"})
		assert.Error(t, err)
	})

	t.Run("AgeEncryptFileMulti with nonexistent plain file returns error", func(t *testing.T) {
		err := AgeEncryptFileMulti(path.Join(test.TestData, "no-such.txt"), cryptedFile, []string{pubA})
		assert.Error(t, err)
	})
}
