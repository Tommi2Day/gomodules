package pwlib

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"

	"github.com/stretchr/testify/require"
)

func TestKMS(t *testing.T) {
	if os.Getenv("SKIP_KMS") != "" {
		t.Skip("Skipping KMS testing in CI environment")
	}
	test.InitTestDirs()

	app := "test_kms_file"
	testdata := test.TestData
	// set env
	pc := NewConfig(app, testdata, testdata, app, typeKMS)
	err := os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	filename := pc.PlainTextFile
	_ = os.Remove(filename)
	//nolint gosec
	err = os.WriteFile(filename, []byte(plainfile), 0644)
	require.NoErrorf(t, err, "Create testdata failed")

	var kmsClient *kms.Client
	kmsContainer, err := prepareKmsContainer()
	require.NoErrorf(t, err, "KMS Server not available")
	require.NotNil(t, kmsContainer, "Prepare failed")
	defer common.DestroyDockerContainer(kmsContainer)

	_ = os.Setenv("AWS_ACCESS_KEY_ID", "abcdef")
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", "abcdefSecret")
	_ = os.Setenv("AWS_DEFAULT_REGION", "eu-central-1")
	_ = os.Setenv("KMS_ENDPOINT", kmsAddress)

	// test

	t.Run("TestKMSConnect", func(t *testing.T) {
		kmsClient = ConnectToKMS()
		require.NotNil(t, kmsClient, "Connect to KMS failed")
	})
	if kmsClient == nil {
		t.Fatal("Connect to KMS failed")
	}

	t.Run("TestKMSListKeys", func(t *testing.T) {
		keys, err := ListKMSKeys(kmsClient)
		require.NoErrorf(t, err, "ListKeys failed:%s", err)
		require.NotNil(t, keys, "ListKeys empty")
		l := len(keys)
		t.Log("Keys found:", l)
		assert.Greater(t, l, 0, "zero Keys returned")
		if l > 0 {
			key := keys[0]
			t.Logf("KeyId=%s, KeyArn=%s", *key.KeyId, *key.KeyArn)
		}
	})
	myKeyID := ""
	t.Run("TestKMSCreateKey", func(t *testing.T) {
		key, err := GenKMSKey(kmsClient, "", "TestKey", map[string]string{"test": "test"})
		require.NoErrorf(t, err, "CreateKeys failed:%s", err)
		require.NotNil(t, key, "createKey returned nil")
		keyID, keyARN := GetKMSKeyIDs(key.KeyMetadata)
		keyDesc := key.KeyMetadata.Description
		t.Logf("KeyId=%s, KeyArn=%s, Desc=%s", keyID, keyARN, *keyDesc)
		assert.NotEmptyf(t, keyID, "KeyID empty")
		assert.NotEmptyf(t, keyARN, "KeyARN empty")
		assert.Equal(t, "TestKey", *keyDesc, "Description not as expected")
		myKeyID = keyID
		pc.KMSKeyID = myKeyID
	})

	if myKeyID == "" {
		t.Fatalf("Key creation failedempty")
		return
	}
	t.Run("DescribeKey", func(t *testing.T) {
		output, err := DescribeKMSKey(kmsClient, myKeyID)
		require.NoErrorf(t, err, "DescribeKey failed:%s", err)
		require.NotNil(t, output, "DescribeKey Result empty")
		keyID, _ := GetKMSKeyIDs(output.KeyMetadata)
		assert.Equal(t, myKeyID, keyID, "KeyID mismatch")
	})
	aliasName := "testkey"
	t.Run("CreateAlias", func(t *testing.T) {
		output, err := CreateKMSAlias(kmsClient, aliasName, myKeyID)
		require.NoErrorf(t, err, "CreateAlias failed:%s", err)
		require.NotNil(t, output, "CreateAlias empty")
	})
	t.Run("ListAliases", func(t *testing.T) {
		aliases, err := ListKMSAliases(kmsClient, "")
		require.NoErrorf(t, err, "ListAliases failed:%s", err)
		l := len(aliases)
		t.Log("Aliases found:", l)
		assert.Greater(t, l, 0, "zero Aliases returned")
		if l > 0 {
			alias := aliases[0]
			an, arn, keyid := GetKMSAliasIDs(&alias)
			assert.NotEmpty(t, keyid)
			t.Logf("AliasName=%s, AliasArn=%s, TargetID=%s", an, arn, keyid)
		}
	})
	t.Run("DescribeAlias", func(t *testing.T) {
		entry, err := DescribeKMSAlias(kmsClient, aliasName)
		require.NoErrorf(t, err, "DescribeAlias failed:%s", err)
		require.NotNil(t, entry, "DescribeAlias empty")
		keyid, _, _ := GetKMSAliasIDs(entry)
		assert.NotEmpty(t, keyid)
		assert.Equal(t, myKeyID, keyid, "KeyID mismatch")
	})

	t.Run("TestEncryptKMSAndDecrypt", func(t *testing.T) {
		enc := ""
		dec := ""
		if myKeyID == "" {
			t.Fatalf("KeyID empty")
		}
		enc, err = KMSEncryptString(kmsClient, myKeyID, plaintext)
		if err != nil {
			t.Fatalf("Test errored at encrypt: %s", err)
		}
		t.Logf("Encrypted: %s", enc)
		dec, err = KMSDecryptString(kmsClient, myKeyID, enc)
		if err != nil {
			t.Fatalf("Test errored at decrypt: %s", err)
		}

		t.Logf("Decrypted: %s", dec)
		if dec != plaintext {
			t.Errorf("Decrypted text did not match input.")
		}
	})

	t.Run("default Encrypt File", func(t *testing.T) {
		err = KMSEncryptFile(pc.PlainTextFile, pc.CryptedFile, myKeyID, pc.SessionPassFile)
		assert.NoErrorf(t, err, "Encryption failed: %s", err)
		assert.FileExists(t, pc.CryptedFile)
	})
	t.Run("default Decrypt File", func(t *testing.T) {
		plaintxt, err := common.ReadFileToString(pc.PlainTextFile)
		require.NoErrorf(t, err, "PlainTextfile %s not readable:%s", pc.PlainTextFile, err)
		expected := len(plaintxt)
		content, err := KMSDecryptFile(pc.CryptedFile, myKeyID, pc.SessionPassFile)
		assert.NoErrorf(t, err, "Decryption failed: %s", err)
		assert.NotEmpty(t, content)
		actual := len(content)
		assert.Equalf(t, expected, actual, "Lines misamtch exp:%d,act:%d", expected, actual)
	})
	t.Run("KMSGetPassword", func(t *testing.T) {
		pass := ""
		pass, err = pc.GetPassword("test", "testuser")
		expected := "testpass"
		assert.NoErrorf(t, err, "Got unexpected error: %s", err)
		assert.Equal(t, expected, pass, "Answer not expected. exp:%s,act:%s", expected, pass)
	})
	t.Run("DeleteAlias", func(t *testing.T) {
		output, err := DeleteKMSAlias(kmsClient, "alias/"+aliasName)
		require.NoErrorf(t, err, "DeleteAlias failed:%s", err)
		require.NotNil(t, output, "DeleteAlias empty")
		aliases, err := ListKMSAliases(kmsClient, myKeyID)
		require.NoErrorf(t, err, "ListAliases after Delete failed:%s", err)
		assert.Zero(t, len(aliases), "ListAliases should be empty after delete")
	})
}
