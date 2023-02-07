package test

import (
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/pwlib"
	"os"
	"testing"

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
	app := "test_encrypt"
	pwlib.SetConfig(app, TestData, TestData, app)

	err := os.Chdir(TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	filename := pwlib.PwConfig.PlainTextFile
	_ = os.Remove(filename)
	//nolint gosec
	err = os.WriteFile(filename, []byte(plain), 0644)
	require.NoErrorf(t, err, "Create testdata failed")
	_, _, err = pwlib.GenRsaKey(pwlib.PwConfig.PubKeyFile, pwlib.PwConfig.PrivateKeyFile, pwlib.PwConfig.KeyPass)
	require.NoErrorf(t, err, "Prepare Key failed:%s", err)

	// run
	t.Run("default Encrypt File", func(t *testing.T) {
		err := pwlib.EncryptFile()
		assert.NoErrorf(t, err, "Encryption failed: %s", err)
		assert.FileExists(t, pwlib.PwConfig.CryptedFile)
	})
	t.Run("default Decrypt File", func(t *testing.T) {
		plain, err := common.ReadFileByLine(pwlib.PwConfig.PlainTextFile)
		require.NoErrorf(t, err, "PlainTextfile %s not readable:%s", err)
		expected := len(plain)
		content, err := pwlib.DecryptFile()
		assert.NoErrorf(t, err, "Decryption failed: %s", err)
		assert.NotEmpty(t, content)
		actual := len(content)
		assert.Equalf(t, expected, actual, "Lines misamtch exp:%d,act:%d", expected, actual)
	})
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
	app := "test_get_pass"
	pwlib.SetConfig(app, TestData, TestData, app)
	err := os.Chdir(TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	filename := pwlib.PwConfig.PlainTextFile
	_ = os.Remove(filename)
	//nolint gosec
	err = os.WriteFile(filename, []byte(plain), 0644)
	require.NoErrorf(t, err, "Create testdata failed")
	_, _, err = pwlib.GenRsaKey(pwlib.PwConfig.PubKeyFile, pwlib.PwConfig.PrivateKeyFile, pwlib.PwConfig.KeyPass)
	require.NoErrorf(t, err, "Prepare Key failed:%s", err)
	err = pwlib.EncryptFile()
	require.NoErrorf(t, err, "Encrypt Plain failed:%s", err)
	_, err = pwlib.ListPasswords()
	require.NoErrorf(t, err, "List failed:%s", err)

	// run
	for _, test := range []testTableType{
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
		t.Run(test.name, func(t *testing.T) {
			pass, err := pwlib.GetPassword(test.system, test.account)
			if test.hasError {
				assert.Error(t, err, "Expected Error not thrown")
			} else {
				assert.NoErrorf(t, err, "Got unexpected error: %s", err)
				assert.Equal(t, test.answer, pass, "Answer not expected. exp:%s,act:%s", test.answer, pass)
			}
		})
	}
}
