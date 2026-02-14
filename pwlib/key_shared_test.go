package pwlib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"
)

func runKeyGenTest(t *testing.T, keyType string, genFunc func(pub, priv, pass string) (any, any, error), expectedPubType any, expectedPrivType any) {
	test.InitTestDirs()
	err := os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")

	pubfilename := "testdata/" + keyType + "_key.pub"
	privfilename := "testdata/" + keyType + "_key.pem"
	_ = os.Remove(pubfilename)
	_ = os.Remove(privfilename)

	t.Run(keyType+" Key Gen unencrypted", func(t *testing.T) {
		pubkey, privkey, err := genFunc(pubfilename, privfilename, "")
		require.NoErrorf(t, err, "Error while creating key: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.NotEmpty(t, privkey)
		assert.IsTypef(t, expectedPubType, pubkey, "Not a %s public key", keyType)
		assert.IsTypef(t, expectedPrivType, privkey, "Not a %s private key", keyType)
		assert.FileExists(t, pubfilename)
		assert.FileExists(t, privfilename)
	})

	pubfilename = "testdata/" + keyType + "_enckey.pub"
	privfilename = "testdata/" + keyType + "_enckey.pem"
	_ = os.Remove(pubfilename)
	_ = os.Remove(privfilename)

	t.Run(keyType+" Key Gen encrypted", func(t *testing.T) {
		pubkey, privkey, err := genFunc(pubfilename, privfilename, "gen_test")
		require.NoErrorf(t, err, "Error while creating key: %s", err)
		assert.NotEmpty(t, pubkey)
		assert.NotEmpty(t, privkey)
		assert.IsTypef(t, expectedPubType, pubkey, "Not a %s public key", keyType)
		assert.IsTypef(t, expectedPrivType, privkey, "Not a %s private key", keyType)
		assert.FileExists(t, pubfilename)
		assert.FileExists(t, privfilename)
		content, err := common.ReadFileToString(privfilename)
		assert.NoErrorf(t, err, "File Read Error %s", err)
		assert.Contains(t, content, "Proc-Type: 4,ENCRYPTED")
	})
}
