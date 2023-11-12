package pwlib

import (
	"os"
	"slices"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const plain = `
# Testfile
!default:defuser2:failure
!default:testuser:default
test:testuser:testpass
testdp:testuser:xxx:yyy
!default:defuser2:default
!default:testuser:failure
!default:defuser:default
`

func TestCrypt(t *testing.T) {
	// prepare
	test.Testinit(t)
	err := os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	dataDir := test.TestData
	keyDir := test.TestData
	var entity *openpgp.Entity
	for _, m := range Methods {
		if slices.Contains([]string{typeVault, typeGopass}, m) {
			continue
		}

		app := "test_encrypt_" + m
		pc := NewConfig(app, dataDir, keyDir, app, m)
		filename := pc.PlainTextFile
		_ = os.Remove(filename)
		//nolint gosec
		err = os.WriteFile(filename, []byte(plain), 0644)
		require.NoErrorf(t, err, "Create testdata failed")

		// genkey or use existing for GPG
		if m == typeGPG {
			entity, _, err = CreateGPGEntity(testGPGName, "TestCrypt", testGPGEmail, pc.KeyPass)
			require.NoErrorf(t, err, "Prepare GPG Keys failed:%s", err)
			err = ExportGPGKeyPair(entity, pc.PubKeyFile, pc.PrivateKeyFile)
			require.NoErrorf(t, err, "Export GPG Keys failed:%s", err)
		} else {
			_, _, err = GenRsaKey(pc.PubKeyFile, pc.PrivateKeyFile, pc.KeyPass)
			require.NoErrorf(t, err, "Prepare Key failed:%s", err)
		}

		// run
		t.Run("Encrypt File method "+m, func(t *testing.T) {
			err = pc.EncryptFile()
			assert.NoErrorf(t, err, "Encryption failed: %s", err)
			assert.FileExists(t, pc.CryptedFile)
		})
		t.Run("Decrypt File method "+m, func(t *testing.T) {
			var plaintxt []string
			plaintxt, err = common.ReadFileByLine(pc.PlainTextFile)
			require.NoErrorf(t, err, "Plain textfile %s not readable:%s", err)
			expected := len(plaintxt)
			content, err := pc.DecryptFile()
			assert.NoErrorf(t, err, "Decryption failed: %s", err)
			assert.NotEmpty(t, content)
			actual := len(content)
			assert.Equalf(t, expected, actual, "Lines misamtch exp:%d,act:%d", expected, actual)
		})
	}
}

func TestGetPassword(t *testing.T) {
	// prepare
	type testTableType struct {
		name     string
		account  string
		system   string
		answer   string
		hasError bool
	}
	test.Testinit(t)
	app := "test_get_pass"
	pc := NewConfig(app, test.TestData, test.TestData, app, typeGO)
	err := os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	filename := pc.PlainTextFile
	_ = os.Remove(filename)
	//nolint gosec
	err = os.WriteFile(filename, []byte(plain), 0644)
	require.NoErrorf(t, err, "Create testdata failed")
	_, _, err = GenRsaKey(pc.PubKeyFile, pc.PrivateKeyFile, pc.KeyPass)
	require.NoErrorf(t, err, "Prepare Key failed:%s", err)
	err = pc.EncryptFile()
	require.NoErrorf(t, err, "Encrypt Plain failed:%s", err)
	_, err = pc.ListPasswords()
	require.NoErrorf(t, err, "List failed:%s", err)

	// run
	for _, testConfig := range []testTableType{
		{
			name:     "direct match",
			account:  "testuser",
			system:   "test",
			answer:   "testpass",
			hasError: false,
		},
		{
			name:     "default 1",
			account:  "defuser",
			system:   "",
			answer:   "default",
			hasError: false,
		},
		{
			name:     "default 2",
			account:  "defuser2",
			system:   "",
			answer:   "default",
			hasError: false,
		},
		{
			name:     "no input",
			account:  "",
			system:   "",
			answer:   "true",
			hasError: true,
		},
		{
			name:     "DP in Password",
			account:  "testuser",
			system:   "testdp",
			answer:   "xxx:yyy",
			hasError: false,
		},
	} {
		t.Run(testConfig.name, func(t *testing.T) {
			pass, err := pc.GetPassword(testConfig.system, testConfig.account)
			if testConfig.hasError {
				assert.Error(t, err, "Expected Error not thrown")
			} else {
				assert.NoErrorf(t, err, "Got unexpected error: %s", err)
				assert.Equal(t, testConfig.answer, pass, "Answer not expected. exp:%s,act:%s", testConfig.answer, pass)
			}
		})
	}
}
