package test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/pwlib"
	"os"
	"os/exec"
	"strings"
	"testing"

	openssl "github.com/Luzifer/go-openssl/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	plaintext = "hallowelt"
	//nolint gosec
	passphrase = "z4yH36a6zerhfE5427ZV"
	plainfile  = `
# Testfile
!default:defuser2:failure
!default:testuser:default
test:testuser:testpass
testdp:testuser:xxx:yyy
!default:defuser2:default
!default:testuser:failure
!default:defuser:default
`
)

// var digest = openssl.BytesToKeySHA256

func TestEncryptToDecrypt(t *testing.T) {
	o := openssl.New()
	enc, err := o.EncryptBytes(passphrase, []byte(plaintext), pwlib.SSLDigest)
	if err != nil {
		t.Fatalf("Test errored at encrypt: %s", err)
	}

	dec, err := o.DecryptBytes(passphrase, enc, pwlib.SSLDigest)
	if err != nil {
		t.Fatalf("Test errored at decrypt: %s", err)
	}

	if string(dec) != plaintext {
		t.Errorf("Decrypted text did not match input.")
	}
}

func TestPublicEncryptString(t *testing.T) {
	app := "test_encrypt_String"
	testdata := TestDir + "/testdata"
	pwlib.SetConfig(app, testdata, testdata, "Test", typeGO)

	err := os.Chdir(TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	_, _, err = pwlib.GenRsaKey(pwlib.PwConfig.PubKeyFile, pwlib.PwConfig.PrivateKeyFile, pwlib.PwConfig.KeyPass)
	require.NoErrorf(t, err, "Prepare Key failed:%s", err)

	crypted, err := pwlib.PublicEncryptString(plaintext, pwlib.PwConfig.PubKeyFile)
	// run
	t.Run("default Encrypt String", func(t *testing.T) {
		assert.NoErrorf(t, err, "Encryption failed: %s", err)
		assert.NotEmpty(t, crypted, "Crypted String empty")
	})

	t.Run("default Decrypt String", func(t *testing.T) {
		actual, err := pwlib.PrivateDecryptString(crypted, pwlib.PwConfig.PrivateKeyFile, pwlib.PwConfig.KeyPass)
		expected := plaintext
		assert.NoErrorf(t, err, "Decryption failed: %s", err)
		assert.NotEmpty(t, actual)
		assert.Equalf(t, expected, actual, "Data Mismatch exp:%s,act:%s", expected, actual)
	})
}

func TestOpensslCompString(t *testing.T) {
	// echo -n "$plain"|openssl rsautl -encrypt -pkcs -inkey $PUBLICKEYFILE -pubin |base64
	// echo -n "$CRYPTED"|base64 -d   |openssl rsautl -decrypt -inkey ${PRIVATEKEYFILE} -passin pass:$KEYPASS

	var cmdout bytes.Buffer
	var cmderr bytes.Buffer
	app := "test_openssl_string"
	testdata := TestDir + "/testdata"

	// set env
	pwlib.SetConfig(app, testdata, testdata, "Test", typeOpenssl)
	err := os.Chdir(TestDir)
	require.NoErrorf(t, err, "ChDir failed")

	// prepare keys
	_, _, err = pwlib.GenRsaKey(pwlib.PwConfig.PubKeyFile, pwlib.PwConfig.PrivateKeyFile, pwlib.PwConfig.KeyPass)
	require.NoErrorf(t, err, "Prepare Key failed:%s", err)
	t.Run("Encrypt_Openssl-Decrypt_String", func(t *testing.T) {
		// encrypt using openssl os cmd
		cmdArgs := []string{
			"openssl", "rsautl",
			"-inkey", pwlib.PwConfig.PubKeyFile,
			"-pubin",
			"-pkcs",
			"-encrypt",
		}
		// nolint gosec
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		t.Logf("CMD: %v", cmdArgs)
		cmdout.Reset()
		cmderr.Reset()
		cmd.Stdout = &cmdout
		cmd.Stderr = &cmderr
		cmd.Stdin = strings.NewReader(plaintext)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Running openssl CLI failed: %v (%s)", err, cmderr.String())
		}
		// encode base64
		crypted := base64.StdEncoding.EncodeToString(cmdout.Bytes())

		// decode openssl encoded string with go functions
		actual, err := pwlib.PrivateDecryptString(crypted, pwlib.PwConfig.PrivateKeyFile, pwlib.PwConfig.KeyPass)
		if err != nil {
			t.Fatalf("Decryprion failed: %v", err)
		}
		// compare
		expected := plaintext
		assert.NotEmpty(t, actual)
		assert.Equalf(t, expected, actual, "Data Mismatch exp:%s,act:%s", expected, actual)
	})

	t.Run("Encrypt_String-OpenSSL_decrypt", func(t *testing.T) {
		// encode string with go functions
		crypted, err := pwlib.PublicEncryptString(plaintext, pwlib.PwConfig.PubKeyFile)
		if err != nil {
			t.Fatalf("Encryprion failed: %v", err)
		}
		t.Logf("B64: %s", crypted)
		// revert base64 encoding
		b64dec, err := base64.StdEncoding.DecodeString(crypted)
		if err != nil {
			t.Fatalf("decode base64 failed: %v", err)
		}

		// decode crypted string in bin format using openssl os cmd
		cmdArgs := []string{
			"openssl", "rsautl",
			"-inkey", pwlib.PwConfig.PrivateKeyFile,
			"-pkcs",
			"-decrypt",
			"-passin", fmt.Sprintf("pass:%s", pwlib.PwConfig.KeyPass),
		}
		// nolint gosec
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		t.Logf("CMD: %v", cmdArgs)
		cmdout.Reset()
		cmderr.Reset()
		cmd.Stdout = &cmdout
		cmd.Stderr = &cmderr
		cmd.Stdin = bytes.NewReader(b64dec)
		expected := plaintext
		if err := cmd.Run(); err != nil {
			t.Fatalf("Running openssl CLI failed: %v (%s)", err, cmderr.String())
		}
		actual := cmdout.String()
		// compare
		assert.NotEmpty(t, actual)
		assert.Equalf(t, expected, actual, "Data Mismatch exp:%s,act:%s", expected, actual)
	})
}

func TestOpensslFile(t *testing.T) {
	var cmdout bytes.Buffer
	var cmderr bytes.Buffer

	app := "test_openssl_file"
	testdata := TestDir + "/testdata"
	// set env
	pwlib.SetConfig(app, testdata, testdata, app, typeOpenssl)
	err := os.Chdir(TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	filename := pwlib.PwConfig.PlainTextFile
	_ = os.Remove(filename)
	//nolint gosec
	err = os.WriteFile(filename, []byte(plainfile), 0644)
	require.NoErrorf(t, err, "Create testdata failed")

	// prepare keys
	_, _, err = pwlib.GenRsaKey(pwlib.PwConfig.PubKeyFile, pwlib.PwConfig.PrivateKeyFile, pwlib.PwConfig.KeyPass)
	require.NoErrorf(t, err, "Prepare Key failed:%s", err)
	// run
	t.Run("default Encrypt File", func(t *testing.T) {
		err := pwlib.PubEncryptFileSSL(pwlib.PwConfig.PlainTextFile, pwlib.PwConfig.CryptedFile, pwlib.PwConfig.PubKeyFile, pwlib.PwConfig.SessionPassFile)
		assert.NoErrorf(t, err, "Encryption failed: %s", err)
		assert.FileExists(t, pwlib.PwConfig.CryptedFile)
	})
	t.Run("default Decrypt File", func(t *testing.T) {
		plaintxt, err := common.ReadFileToString(pwlib.PwConfig.PlainTextFile)
		require.NoErrorf(t, err, "PlainTextfile %s not readable:%s", err)
		expected := len(plaintxt)
		content, err := pwlib.PrivateDecryptFileSSL(pwlib.PwConfig.CryptedFile, pwlib.PwConfig.PrivateKeyFile, pwlib.PwConfig.KeyPass, pwlib.PwConfig.SessionPassFile)
		assert.NoErrorf(t, err, "Decryption failed: %s", err)
		assert.NotEmpty(t, content)
		actual := len(content)
		assert.Equalf(t, expected, actual, "Lines misamtch exp:%d,act:%d", expected, actual)
	})
	t.Run("Encrypt_Openssl-Decrypt_Api", func(t *testing.T) {
		const rb = 16
		var actual, crypted string
		// create session key
		random := make([]byte, rb)
		_, err = rand.Read(random)
		if err != nil {
			t.Fatalf("Cannot generate session key:%s", err)
		}
		sessionKey := base64.StdEncoding.EncodeToString(random)
		t.Logf("Create Random SessionKeyin  %s: %s", pwlib.PwConfig.SessionPassFile, sessionKey)

		// encrypt session key and save to file
		// echo -n sessionKey |openssl rsautl -encrypt -pkcs -inkey PubKeyFile -pubin |openssl enc -base64 -out SessionPassFile
		crypted, err = pwlib.PublicEncryptString(sessionKey, pwlib.PwConfig.PubKeyFile)
		if err != nil {
			t.Fatalf("Encrypting Keyfile failed: %v", err)
		}
		//nolint gosec
		err = os.WriteFile(pwlib.PwConfig.SessionPassFile, []byte(crypted), 0644)
		if err != nil {
			t.Fatalf("Cannot write session Key file %s:%v", pwlib.PwConfig.SessionPassFile, err)
		}

		// encrypt using openssl cmd
		cmdArgs := []string{
			"openssl", "enc", "-e",
			"-aes-256-cbc",
			"-base64",
			"-pass", fmt.Sprintf("pass:%s", sessionKey),
			"-md", "sha256",
			"-in", pwlib.PwConfig.PlainTextFile,
			"-out", pwlib.PwConfig.CryptedFile,
		}
		// nolint gosec
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		t.Logf("CMD: %v", cmdArgs)
		cmdout.Reset()
		cmderr.Reset()
		cmd.Stdout = &cmdout
		cmd.Stderr = &cmderr
		if err = cmd.Run(); err != nil {
			t.Fatalf("Running openssl CLI failed: %v (%s)", err, cmderr.String())
		}

		// decrypt openssl encoded data using API
		actual, err = pwlib.PrivateDecryptFileSSL(pwlib.PwConfig.CryptedFile, pwlib.PwConfig.PrivateKeyFile, pwlib.PwConfig.KeyPass, pwlib.PwConfig.SessionPassFile)

		// compare
		expected := plainfile
		assert.NotEmpty(t, actual)
		assert.Equalf(t, expected, actual, "Data Mismatch exp:%s,act:%s", expected, actual)
	})
	t.Run("Encrypt_API-Decrypt_openssl", func(t *testing.T) {
		// encrypt using api
		err := pwlib.PubEncryptFileSSL(pwlib.PwConfig.PlainTextFile, pwlib.PwConfig.CryptedFile, pwlib.PwConfig.PubKeyFile, pwlib.PwConfig.SessionPassFile)
		assert.NoErrorf(t, err, "Cannot Encrypt using API:%s", err)
		if err != nil {
			t.Fatalf("Cannot Encrypt using API:%s", err)
		}

		// verify witch openssl cmd
		// read session pass file
		//nolint gosec
		data, err := os.ReadFile(pwlib.PwConfig.SessionPassFile)
		if err != nil {
			t.Fatalf("Cannot Read SessionPassFile %s:%v", pwlib.PwConfig.SessionPassFile, err)
		}
		cryptedKey := string(data)
		// revert base64 encoding
		b64dec, err := base64.StdEncoding.DecodeString(cryptedKey)
		if err != nil {
			t.Fatalf("decode base64 failed: %v", err)
		}

		// decode crypted string in bin format using openssl os cmd
		cmdArgs := []string{
			"openssl", "rsautl",
			"-inkey", pwlib.PwConfig.PrivateKeyFile,
			"-pkcs",
			"-decrypt",
			"-passin", fmt.Sprintf("pass:%s", pwlib.PwConfig.KeyPass),
		}
		// nolint gosec
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		t.Logf("CMD: %v", cmdArgs)
		cmdout.Reset()
		cmderr.Reset()
		cmd.Stdout = &cmdout
		cmd.Stderr = &cmderr
		cmd.Stdin = bytes.NewReader(b64dec)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Running openssl CLI failed: %v (%s)", err, cmderr.String())
		}
		sessionKey := cmdout.String()
		t.Logf("SessionKey: %s", sessionKey)

		// decrypt using openssl cmd, must use -base64 -A for singleline base64 string
		cmdArgs = []string{
			"openssl", "enc", "-d",
			"-aes-256-cbc",
			"-base64",
			"-A",
			"-pass", fmt.Sprintf("pass:%s", sessionKey),
			"-md", "sha256",
			"-in", pwlib.PwConfig.CryptedFile,
		}
		// nolint gosec
		cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
		t.Logf("CMD: %v", cmdArgs)
		cmdout.Reset()
		cmderr.Reset()
		cmd.Stdout = &cmdout
		cmd.Stderr = &cmderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("Running openssl CLI failed: %v (%s)", err, cmderr.String())
		}

		actual := cmdout.String()
		// compare
		expected := plainfile
		assert.NotEmpty(t, actual)
		assert.Equalf(t, expected, actual, "Data Mismatch exp:%s,act:%s", expected, actual)
	})
}
