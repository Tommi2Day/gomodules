package pwlib

import (
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"
)

const (
	gopassTestSecret        = "test/myservice"
	gopassTestNestedSecret  = "databases/prod/postgres"
	gopassTestPassword      = "s3cr3tP@ssw0rd!"
	gopassTestNestedContent = "postgrespass\nusername: postgres\nhost: db.example.com"
)

// setupGopassGPGKeys creates a fresh GPG key pair for gopass tests.
func setupGopassGPGKeys(t *testing.T) (pubKeyFile, privKeyFile string) {
	t.Helper()
	pubKeyFile = filepath.Join(test.TestData, "gopass_test"+pubGPGExt)
	privKeyFile = filepath.Join(test.TestData, "gopass_test"+privGPGExt)
	_ = os.Remove(pubKeyFile)
	_ = os.Remove(privKeyFile)
	entity, _, err := CreateGPGEntity(testGPGName, "GopassTest", testGPGEmail, testGPGPass)
	require.NoErrorf(t, err, "GPG key creation failed: %v", err)
	err = ExportGPGKeyPair(entity, pubKeyFile, privKeyFile)
	require.NoErrorf(t, err, "GPG key export failed: %v", err)
	return
}

// setupGopassAgeKeys creates a fresh age key pair for gopass tests.
func setupGopassAgeKeys(t *testing.T) (pubKeyFile, privKeyFile string) {
	t.Helper()
	pubKeyFile = filepath.Join(test.TestData, "gopass_test"+pubAgeExt)
	privKeyFile = filepath.Join(test.TestData, "gopass_test"+privAgeExt)
	_ = os.Remove(pubKeyFile)
	_ = os.Remove(privKeyFile)
	identity, _, err := CreateAgeIdentity()
	require.NoErrorf(t, err, "age key creation failed: %v", err)
	err = ExportAgeKeyPair(identity, pubKeyFile, privKeyFile)
	require.NoErrorf(t, err, "age key export failed: %v", err)
	return
}

func TestGopassReadSecretLines(t *testing.T) {
	test.InitTestDirs()

	storeDir := filepath.Join(test.TestData, "gopass-lines-store")
	_ = os.RemoveAll(storeDir)
	require.NoError(t, os.MkdirAll(storeDir, 0700))

	pubKeyFile, privKeyFile := setupGopassGPGKeys(t)

	t.Run("password-only secret produces single line", func(t *testing.T) {
		require.NoError(t, GopassWrite(storeDir, "simple/secret", "mypassword", pubKeyFile, GopassCryptoGPG))
		content, err := GopassReadSecretLines(storeDir, "simple/secret", privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.NoError(t, err)
		assert.Equal(t, "simple/secret:password:mypassword", content)
	})

	t.Run("secret with YAML metadata produces multiple lines", func(t *testing.T) {
		raw := "dbpassword\n---\nusername: dbuser\nhost: db.example.com"
		require.NoError(t, GopassWrite(storeDir, "db/postgres", raw, pubKeyFile, GopassCryptoGPG))
		content, err := GopassReadSecretLines(storeDir, "db/postgres", privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.NoError(t, err)
		assert.Contains(t, content, "db/postgres:password:dbpassword")
		assert.Contains(t, content, "db/postgres:username:dbuser")
		assert.Contains(t, content, "db/postgres:host:db.example.com")
	})

	t.Run("free-text lines before YAML separator are ignored", func(t *testing.T) {
		raw := "s3cr3t\nnote: some free text\n---\nurl: https://example.com"
		require.NoError(t, GopassWrite(storeDir, "app/withtext", raw, pubKeyFile, GopassCryptoGPG))
		content, err := GopassReadSecretLines(storeDir, "app/withtext", privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.NoError(t, err)
		assert.Contains(t, content, "app/withtext:password:s3cr3t")
		assert.Contains(t, content, "app/withtext:url:https://example.com")
		assert.NotContains(t, content, "note:")
	})

	t.Run("nonexistent secret returns error", func(t *testing.T) {
		_, err := GopassReadSecretLines(storeDir, "no/such", privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.Error(t, err)
	})

	t.Run("age-encrypted secret also works", func(t *testing.T) {
		agePub := filepath.Join(test.TestData, "lines_age"+pubAgeExt)
		agePriv := filepath.Join(test.TestData, "lines_age"+privAgeExt)
		id, _, err := CreateAgeIdentity()
		require.NoError(t, err)
		require.NoError(t, ExportAgeKeyPair(id, agePub, agePriv))

		raw := "agepass\n---\nowner: alice"
		require.NoError(t, GopassWrite(storeDir, "age/secret", raw, agePub, GopassCryptoAge))
		content, err := GopassReadSecretLines(storeDir, "age/secret", agePriv, "", GopassCryptoAge)
		assert.NoError(t, err)
		assert.Contains(t, content, "age/secret:password:agepass")
		assert.Contains(t, content, "age/secret:owner:alice")
	})
}

func TestGopassGetPassword(t *testing.T) {
	test.InitTestDirs()

	storeDir := filepath.Join(test.TestData, "gopass-getpw-store")
	_ = os.RemoveAll(storeDir)
	require.NoError(t, os.MkdirAll(storeDir, 0700))

	pubKeyFile, privKeyFile := setupGopassGPGKeys(t)

	// place .gpg-id marker so GopassDetectCrypto finds GPG without scanning
	require.NoError(t, common.WriteStringToFile(filepath.Join(storeDir, gopassGPGMarker), "test\n"))

	// write test secrets
	require.NoError(t, GopassWrite(storeDir, "app/myservice", "s3cr3t", pubKeyFile, GopassCryptoGPG))
	require.NoError(t, GopassWrite(storeDir, "databases/prod/postgres",
		"dbpassword\n---\nusername: dbuser\nhost: db.example.com",
		pubKeyFile, GopassCryptoGPG))

	// NewConfig: dataDir = storeDir, override PrivateKeyFile since ext is "" for gopass
	pc := NewConfig("gopass-test", storeDir, storeDir, testGPGPass, typeGopass)
	pc.PrivateKeyFile = privKeyFile

	t.Run("get password field from simple secret", func(t *testing.T) {
		pass, err := pc.GetPassword("app/myservice", "password")
		assert.NoError(t, err)
		assert.Equal(t, "s3cr3t", pass)
	})

	t.Run("get password field from secret with metadata", func(t *testing.T) {
		pass, err := pc.GetPassword("databases/prod/postgres", "password")
		assert.NoError(t, err)
		assert.Equal(t, "dbpassword", pass)
	})

	t.Run("get YAML metadata field", func(t *testing.T) {
		pass, err := pc.GetPassword("databases/prod/postgres", "username")
		assert.NoError(t, err)
		assert.Equal(t, "dbuser", pass)
	})

	t.Run("get another YAML metadata field", func(t *testing.T) {
		pass, err := pc.GetPassword("databases/prod/postgres", "host")
		assert.NoError(t, err)
		assert.Equal(t, "db.example.com", pass)
	})

	t.Run("nonexistent secret path returns error", func(t *testing.T) {
		pass, err := pc.GetPassword("no/such/secret", "password")
		assert.Error(t, err)
		assert.Empty(t, pass)
	})

	t.Run("nonexistent field in existing secret returns error", func(t *testing.T) {
		pass, err := pc.GetPassword("app/myservice", "username")
		assert.Error(t, err)
		assert.Empty(t, pass)
	})

	t.Run("field name matching is case-sensitive (typeGopass sets CaseSensitive)", func(t *testing.T) {
		// "Password" (capital P) must not match the stored field name "password"
		pass, err := pc.GetPassword("app/myservice", "Password")
		assert.Error(t, err)
		assert.Empty(t, pass)
	})

	t.Run("age store also works end-to-end via GetPassword", func(t *testing.T) {
		ageStoreDir := filepath.Join(test.TestData, "gopass-getpw-age-store")
		_ = os.RemoveAll(ageStoreDir)
		require.NoError(t, os.MkdirAll(ageStoreDir, 0700))

		agePub := filepath.Join(test.TestData, "getpw_age"+pubAgeExt)
		agePriv := filepath.Join(test.TestData, "getpw_age"+privAgeExt)
		id, _, err := CreateAgeIdentity()
		require.NoError(t, err)
		require.NoError(t, ExportAgeKeyPair(id, agePub, agePriv))

		require.NoError(t, common.WriteStringToFile(
			filepath.Join(ageStoreDir, gopassAgeMarker), "age1test\n"))
		require.NoError(t, GopassWrite(ageStoreDir, "svc/api", "agetoken\n---\nowner: bob",
			agePub, GopassCryptoAge))

		pcAge := NewConfig("gopass-age-test", ageStoreDir, ageStoreDir, "", typeGopass)
		pcAge.PrivateKeyFile = agePriv

		pass, err := pcAge.GetPassword("svc/api", "password")
		assert.NoError(t, err)
		assert.Equal(t, "agetoken", pass)

		pass, err = pcAge.GetPassword("svc/api", "owner")
		assert.NoError(t, err)
		assert.Equal(t, "bob", pass)
	})
}

func TestGopassAgeEncryptedIdentity(t *testing.T) {
	test.InitTestDirs()

	storeDir := filepath.Join(test.TestData, "gopass-enc-age-store")
	_ = os.RemoveAll(storeDir)
	require.NoError(t, os.MkdirAll(storeDir, 0700))

	pubKeyFile := filepath.Join(test.TestData, "gopass_enc_age"+pubAgeExt)
	encPrivKeyFile := filepath.Join(test.TestData, "gopass_enc_age"+privAgeExt)
	_ = os.Remove(pubKeyFile)
	_ = os.Remove(encPrivKeyFile)

	identity, _, err := CreateAgeIdentity()
	require.NoError(t, err)
	err = ExportAgeKeyPairEncrypted(identity, pubKeyFile, encPrivKeyFile, testAgePassphrase)
	require.NoErrorf(t, err, "ExportAgeKeyPairEncrypted failed: %v", err)

	const secretName = "enc/mysecret"
	const secretValue = "encrypted-identity-password"
	const nestedName = "enc/nested/service"
	const nestedContent = "nestedpass\n---\nowner: alice"

	t.Run("GopassWrite with public key succeeds (public key is always plaintext)", func(t *testing.T) {
		err = GopassWrite(storeDir, secretName, secretValue, pubKeyFile, GopassCryptoAge)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(storeDir, filepath.FromSlash(secretName)+gopassAgeExt))
	})

	t.Run("GopassWrite nested path", func(t *testing.T) {
		err = GopassWrite(storeDir, nestedName, nestedContent, pubKeyFile, GopassCryptoAge)
		assert.NoError(t, err)
	})

	t.Run("GopassRead with encrypted identity and correct passphrase", func(t *testing.T) {
		secret, readErr := GopassRead(storeDir, secretName, encPrivKeyFile, testAgePassphrase, GopassCryptoAge)
		assert.NoError(t, readErr)
		assert.Equal(t, secretValue, secret)
	})

	t.Run("GopassReadRaw returns full content with encrypted identity", func(t *testing.T) {
		raw, readErr := GopassReadRaw(storeDir, nestedName, encPrivKeyFile, testAgePassphrase, GopassCryptoAge)
		assert.NoError(t, readErr)
		assert.Contains(t, raw, "nestedpass")
		assert.Contains(t, raw, "owner: alice")
	})

	t.Run("GopassRead with wrong passphrase returns error", func(t *testing.T) {
		_, readErr := GopassRead(storeDir, secretName, encPrivKeyFile, "wrongpassphrase", GopassCryptoAge)
		assert.Error(t, readErr)
	})

	t.Run("GopassRead with plaintext identity file fails on encrypted identity file", func(t *testing.T) {
		// encPrivKeyFile is encrypted - using it without a passphrase should fail
		// because age.ParseIdentities cannot parse binary age data
		_, readErr := GopassRead(storeDir, secretName, encPrivKeyFile, "", GopassCryptoAge)
		assert.Error(t, readErr)
	})

	t.Run("GopassReadSecretLines with encrypted identity", func(t *testing.T) {
		content, readErr := GopassReadSecretLines(storeDir, nestedName, encPrivKeyFile, testAgePassphrase, GopassCryptoAge)
		assert.NoError(t, readErr)
		assert.Contains(t, content, "enc/nested/service:password:nestedpass")
		assert.Contains(t, content, "enc/nested/service:owner:alice")
	})

	t.Run("GopassList is unaffected by identity encryption", func(t *testing.T) {
		secrets, listErr := GopassList(storeDir, GopassCryptoAge)
		assert.NoError(t, listErr)
		assert.Len(t, secrets, 2)
		assert.Contains(t, secrets, secretName)
		assert.Contains(t, secrets, nestedName)
	})
}

func TestGopassAgePasswordEnv(t *testing.T) {
	test.InitTestDirs()

	storeDir := filepath.Join(test.TestData, "gopass-age-env-store")
	_ = os.RemoveAll(storeDir)
	require.NoError(t, os.MkdirAll(storeDir, 0700))

	// create an encrypted age identity (requires a passphrase to use)
	pubKeyFile := filepath.Join(test.TestData, "gopass_env_age"+pubAgeExt)
	encPrivKeyFile := filepath.Join(test.TestData, "gopass_env_age"+privAgeExt)
	identity, _, err := CreateAgeIdentity()
	require.NoError(t, err)
	require.NoError(t, ExportAgeKeyPairEncrypted(identity, pubKeyFile, encPrivKeyFile, testAgePassphrase))
	require.NoError(t, GopassWrite(storeDir, "env/secret", "envpassword", pubKeyFile, GopassCryptoAge))

	t.Run("GOPASS_AGE_PASSWORD used when keypass is empty", func(t *testing.T) {
		_ = os.Setenv(gopassEnvAgePassword, testAgePassphrase)
		secret, readErr := GopassRead(storeDir, "env/secret", encPrivKeyFile, "", GopassCryptoAge)
		_ = os.Unsetenv(gopassEnvAgePassword)
		assert.NoError(t, readErr)
		assert.Equal(t, "envpassword", secret)
	})

	t.Run("explicit keypass takes precedence over env var", func(t *testing.T) {
		_ = os.Setenv(gopassEnvAgePassword, "wrongpassphrase")
		secret, readErr := GopassRead(storeDir, "env/secret", encPrivKeyFile, testAgePassphrase, GopassCryptoAge)
		_ = os.Unsetenv(gopassEnvAgePassword)
		assert.NoError(t, readErr)
		assert.Equal(t, "envpassword", secret)
	})

	t.Run("wrong env passphrase returns error", func(t *testing.T) {
		_ = os.Setenv(gopassEnvAgePassword, "wrongpassphrase")
		_, readErr := GopassRead(storeDir, "env/secret", encPrivKeyFile, "", GopassCryptoAge)
		_ = os.Unsetenv(gopassEnvAgePassword)
		assert.Error(t, readErr)
	})

	t.Run("no env and no keypass returns error for encrypted identity", func(t *testing.T) {
		_ = os.Unsetenv(gopassEnvAgePassword)
		_, readErr := GopassRead(storeDir, "env/secret", encPrivKeyFile, "", GopassCryptoAge)
		assert.Error(t, readErr)
	})
}

func TestGopassStoreDir(t *testing.T) {
	test.InitTestDirs()
	storeDir := filepath.Join(test.TestData, "gopass-storedir-test")

	t.Run("explicit path returned as-is", func(t *testing.T) {
		dir, err := GopassStoreDir(storeDir)
		assert.NoError(t, err)
		assert.Equal(t, storeDir, dir)
	})

	t.Run("env var takes precedence over default", func(t *testing.T) {
		_ = os.Setenv(gopassEnvStoreDir, storeDir)
		dir, err := GopassStoreDir("")
		assert.NoError(t, err)
		assert.Equal(t, storeDir, dir)
		_ = os.Unsetenv(gopassEnvStoreDir)
	})

	t.Run("default resolves to gopass path under home", func(t *testing.T) {
		_ = os.Unsetenv(gopassEnvStoreDir)
		_ = os.Unsetenv(gopassEnvConfig)
		_ = os.Unsetenv(gopassEnvHomeDir)
		_ = os.Unsetenv(gopassEnvXDGData)
		dir, err := GopassStoreDir("")
		assert.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "gopass")
	})

	t.Run("config root.path used when no env var set", func(t *testing.T) {
		_ = os.Unsetenv(gopassEnvStoreDir)
		cfgFile := filepath.Join(test.TestData, "gopass_storeDir_cfg.yml")
		require.NoError(t, os.WriteFile(cfgFile, []byte("[mounts]\n\tpath = "+storeDir+"\n"), 0600))
		_ = os.Setenv(gopassEnvConfig, cfgFile)
		dir, err := GopassStoreDir("")
		_ = os.Unsetenv(gopassEnvConfig)
		assert.NoError(t, err)
		assert.Equal(t, storeDir, dir)
	})

	t.Run("XDG_DATA_HOME used for default store path", func(t *testing.T) {
		_ = os.Unsetenv(gopassEnvStoreDir)
		_ = os.Setenv(gopassEnvConfig, filepath.Join(test.TestData, "no_such_gopass_cfg.yml"))
		_ = os.Setenv(gopassEnvXDGData, "/custom/data")
		dir, err := GopassStoreDir("")
		_ = os.Unsetenv(gopassEnvConfig)
		_ = os.Unsetenv(gopassEnvXDGData)
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join("/custom/data", "gopass", "stores", "root"), dir)
	})

	t.Run("GOPASS_HOMEDIR used for default store path", func(t *testing.T) {
		_ = os.Unsetenv(gopassEnvStoreDir)
		_ = os.Unsetenv(gopassEnvXDGData)
		_ = os.Setenv(gopassEnvConfig, filepath.Join(test.TestData, "no_such_gopass_cfg.yml"))
		_ = os.Setenv(gopassEnvHomeDir, "/custom/home")
		dir, err := GopassStoreDir("")
		_ = os.Unsetenv(gopassEnvConfig)
		_ = os.Unsetenv(gopassEnvHomeDir)
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join("/custom/home", ".local", "share", "gopass", "stores", "root"), dir)
	})
}

func TestGopassMounts(t *testing.T) {
	test.InitTestDirs()
	cfgFile := filepath.Join(test.TestData, "gopass_mounts_cfg.yml")

	t.Run("root and mounts returned", func(t *testing.T) {
		content := sampleGopassConfig("/stores/root", "/stores/work")
		require.NoError(t, os.WriteFile(cfgFile, []byte(content), 0600))
		stores, err := GopassMounts(cfgFile)
		assert.NoError(t, err)
		require.Contains(t, stores, "root")
		assert.Equal(t, "/stores/root", stores["root"].Path)
		require.Contains(t, stores, "work")
		assert.Equal(t, "/stores/work", stores["work"].Path)
	})

	t.Run("root without path is omitted", func(t *testing.T) {
		require.NoError(t, os.WriteFile(cfgFile, []byte("[mounts \"work\"]\n\tpath = /stores/work\n"), 0600))
		stores, err := GopassMounts(cfgFile)
		assert.NoError(t, err)
		assert.NotContains(t, stores, "root")
		require.Contains(t, stores, "work")
	})

	t.Run("missing config returns error", func(t *testing.T) {
		_, err := GopassMounts(filepath.Join(test.TestData, "no_such_mounts_cfg.yml"))
		assert.Error(t, err)
	})
}

func TestGopassGPG(t *testing.T) {
	test.InitTestDirs()

	storeDir := filepath.Join(test.TestData, "gopass-gpg-store")
	_ = os.RemoveAll(storeDir)
	require.NoError(t, os.MkdirAll(storeDir, 0700))

	pubKeyFile, privKeyFile := setupGopassGPGKeys(t)

	t.Run("GopassList empty store", func(t *testing.T) {
		secrets, err := GopassList(storeDir, GopassCryptoGPG)
		assert.NoError(t, err)
		assert.Empty(t, secrets)
	})

	t.Run("GopassWrite", func(t *testing.T) {
		err := GopassWrite(storeDir, gopassTestSecret, gopassTestPassword, pubKeyFile, GopassCryptoGPG)
		assert.NoErrorf(t, err, "GopassWrite failed: %v", err)
		assert.FileExists(t, filepath.Join(storeDir, filepath.FromSlash(gopassTestSecret)+gopassSecretExt))
	})

	t.Run("GopassWrite nested path", func(t *testing.T) {
		err := GopassWrite(storeDir, gopassTestNestedSecret, gopassTestNestedContent, pubKeyFile, GopassCryptoGPG)
		assert.NoErrorf(t, err, "GopassWrite nested failed: %v", err)
		assert.FileExists(t, filepath.Join(storeDir, filepath.FromSlash(gopassTestNestedSecret)+gopassSecretExt))
	})

	t.Run("GopassList after writes", func(t *testing.T) {
		secrets, err := GopassList(storeDir, GopassCryptoGPG)
		assert.NoErrorf(t, err, "GopassList failed: %v", err)
		assert.Len(t, secrets, 2)
		assert.Contains(t, secrets, gopassTestSecret)
		assert.Contains(t, secrets, gopassTestNestedSecret)
	})

	t.Run("GopassList via env", func(t *testing.T) {
		_ = os.Setenv(gopassEnvStoreDir, storeDir)
		secrets, err := GopassList("", GopassCryptoGPG)
		assert.NoErrorf(t, err, "GopassList via env failed: %v", err)
		assert.Len(t, secrets, 2)
		_ = os.Unsetenv(gopassEnvStoreDir)
	})

	t.Run("GopassRead password", func(t *testing.T) {
		secret, err := GopassRead(storeDir, gopassTestSecret, privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.NoErrorf(t, err, "GopassRead failed: %v", err)
		assert.Equal(t, gopassTestPassword, secret)
	})

	t.Run("GopassRead returns first line only", func(t *testing.T) {
		secret, err := GopassRead(storeDir, gopassTestNestedSecret, privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.NoErrorf(t, err, "GopassRead failed: %v", err)
		assert.Equal(t, "postgrespass", secret)
	})

	t.Run("GopassReadRaw returns full content", func(t *testing.T) {
		raw, err := GopassReadRaw(storeDir, gopassTestNestedSecret, privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.NoErrorf(t, err, "GopassReadRaw failed: %v", err)
		assert.Contains(t, raw, "postgrespass")
		assert.Contains(t, raw, "username: postgres")
		assert.Contains(t, raw, "host: db.example.com")
	})

	t.Run("GopassWrite overwrite existing", func(t *testing.T) {
		const updatedPassword = "newP@ssw0rd"
		err := GopassWrite(storeDir, gopassTestSecret, updatedPassword, pubKeyFile, GopassCryptoGPG)
		require.NoErrorf(t, err, "GopassWrite overwrite failed: %v", err)
		secret, err := GopassRead(storeDir, gopassTestSecret, privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.NoError(t, err)
		assert.Equal(t, updatedPassword, secret)
	})

	t.Run("GetGopassSecret nested path", func(t *testing.T) {
		content, err := GetGopassSecret(storeDir, gopassTestNestedSecret, privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.NoErrorf(t, err, "GetGopassSecret failed: %v", err)
		assert.Contains(t, content, "databases/prod:postgres:postgrespass")
	})

	t.Run("GetGopassSecret flat path", func(t *testing.T) {
		err := GopassWrite(storeDir, "toplevelsecret", "toplevelpass", pubKeyFile, GopassCryptoGPG)
		require.NoError(t, err)
		content, err := GetGopassSecret(storeDir, "toplevelsecret", privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.NoErrorf(t, err, "GetGopassSecret failed: %v", err)
		assert.Contains(t, content, "toplevelsecret:toplevelsecret:toplevelpass")
	})

	t.Run("GopassRead nonexistent secret", func(t *testing.T) {
		_, err := GopassRead(storeDir, "nonexistent/secret", privKeyFile, testGPGPass, GopassCryptoGPG)
		assert.Error(t, err)
	})

	t.Run("GopassRead wrong key file", func(t *testing.T) {
		_, err := GopassRead(storeDir, gopassTestSecret, "/nonexistent/key.gpg", testGPGPass, GopassCryptoGPG)
		assert.Error(t, err)
	})

	t.Run("GopassRead wrong passphrase", func(t *testing.T) {
		_, err := GopassRead(storeDir, gopassTestSecret, privKeyFile, "wrongpassphrase", GopassCryptoGPG)
		assert.Error(t, err)
	})
}

func TestGopassAge(t *testing.T) {
	test.InitTestDirs()

	storeDir := filepath.Join(test.TestData, "gopass-age-store")
	_ = os.RemoveAll(storeDir)
	require.NoError(t, os.MkdirAll(storeDir, 0700))

	pubKeyFile, privKeyFile := setupGopassAgeKeys(t)

	t.Run("GopassList empty store", func(t *testing.T) {
		secrets, err := GopassList(storeDir, GopassCryptoAge)
		assert.NoError(t, err)
		assert.Empty(t, secrets)
	})

	t.Run("GopassWrite", func(t *testing.T) {
		err := GopassWrite(storeDir, gopassTestSecret, gopassTestPassword, pubKeyFile, GopassCryptoAge)
		assert.NoErrorf(t, err, "GopassWrite age failed: %v", err)
		assert.FileExists(t, filepath.Join(storeDir, filepath.FromSlash(gopassTestSecret)+gopassAgeExt))
	})

	t.Run("GopassWrite nested path", func(t *testing.T) {
		err := GopassWrite(storeDir, gopassTestNestedSecret, gopassTestNestedContent, pubKeyFile, GopassCryptoAge)
		assert.NoErrorf(t, err, "GopassWrite age nested failed: %v", err)
		assert.FileExists(t, filepath.Join(storeDir, filepath.FromSlash(gopassTestNestedSecret)+gopassAgeExt))
	})

	t.Run("GopassList after writes", func(t *testing.T) {
		secrets, err := GopassList(storeDir, GopassCryptoAge)
		assert.NoErrorf(t, err, "GopassList age failed: %v", err)
		assert.Len(t, secrets, 2)
		assert.Contains(t, secrets, gopassTestSecret)
		assert.Contains(t, secrets, gopassTestNestedSecret)
	})

	t.Run("GopassRead password", func(t *testing.T) {
		secret, err := GopassRead(storeDir, gopassTestSecret, privKeyFile, "", GopassCryptoAge)
		assert.NoErrorf(t, err, "GopassRead age failed: %v", err)
		assert.Equal(t, gopassTestPassword, secret)
	})

	t.Run("GopassRead returns first line only", func(t *testing.T) {
		secret, err := GopassRead(storeDir, gopassTestNestedSecret, privKeyFile, "", GopassCryptoAge)
		assert.NoErrorf(t, err, "GopassRead age failed: %v", err)
		assert.Equal(t, "postgrespass", secret)
	})

	t.Run("GopassReadRaw returns full content", func(t *testing.T) {
		raw, err := GopassReadRaw(storeDir, gopassTestNestedSecret, privKeyFile, "", GopassCryptoAge)
		assert.NoErrorf(t, err, "GopassReadRaw age failed: %v", err)
		assert.Contains(t, raw, "postgrespass")
		assert.Contains(t, raw, "username: postgres")
		assert.Contains(t, raw, "host: db.example.com")
	})

	t.Run("GopassWrite overwrite existing", func(t *testing.T) {
		const updatedPassword = "newAgeP@ss"
		err := GopassWrite(storeDir, gopassTestSecret, updatedPassword, pubKeyFile, GopassCryptoAge)
		require.NoErrorf(t, err, "GopassWrite age overwrite failed: %v", err)
		secret, err := GopassRead(storeDir, gopassTestSecret, privKeyFile, "", GopassCryptoAge)
		assert.NoError(t, err)
		assert.Equal(t, updatedPassword, secret)
	})

	t.Run("GetGopassSecret nested path", func(t *testing.T) {
		content, err := GetGopassSecret(storeDir, gopassTestNestedSecret, privKeyFile, "", GopassCryptoAge)
		assert.NoErrorf(t, err, "GetGopassSecret age failed: %v", err)
		assert.Contains(t, content, "databases/prod:postgres:postgrespass")
	})

	t.Run("GetGopassSecret flat path", func(t *testing.T) {
		err := GopassWrite(storeDir, "toplevelsecret", "toplevelpass", pubKeyFile, GopassCryptoAge)
		require.NoError(t, err)
		content, err := GetGopassSecret(storeDir, "toplevelsecret", privKeyFile, "", GopassCryptoAge)
		assert.NoErrorf(t, err, "GetGopassSecret age failed: %v", err)
		assert.Contains(t, content, "toplevelsecret:toplevelsecret:toplevelpass")
	})

	t.Run("GopassRead nonexistent secret", func(t *testing.T) {
		_, err := GopassRead(storeDir, "nonexistent/secret", privKeyFile, "", GopassCryptoAge)
		assert.Error(t, err)
	})

	t.Run("GopassRead wrong identity file", func(t *testing.T) {
		_, err := GopassRead(storeDir, gopassTestSecret, "/nonexistent/identity.age", "", GopassCryptoAge)
		assert.Error(t, err)
	})

	t.Run("GopassRead with wrong identity key", func(t *testing.T) {
		// generate a different age identity — it should not decrypt secrets encrypted to the first one
		otherIdentity, _, err := CreateAgeIdentity()
		require.NoError(t, err)
		otherPrivFile := filepath.Join(test.TestData, "gopass_other"+privAgeExt)
		otherPubFile := filepath.Join(test.TestData, "gopass_other"+pubAgeExt)
		defer func() {
			_ = os.Remove(otherPrivFile)
			_ = os.Remove(otherPubFile)
		}()
		err = ExportAgeKeyPair(otherIdentity, otherPubFile, otherPrivFile)
		require.NoError(t, err)
		_, err = GopassRead(storeDir, gopassTestSecret, otherPrivFile, "", GopassCryptoAge)
		assert.Error(t, err)
	})

	// plaintext identity: keypass is unused when identity parses as a plaintext age key
	t.Run("GopassRead age plaintext identity with non-empty keypass", func(t *testing.T) {
		secret, err := GopassRead(storeDir, gopassTestNestedSecret, privKeyFile, "not-a-real-passphrase", GopassCryptoAge)
		assert.NoError(t, err)
		assert.Equal(t, "postgrespass", secret)
	})
}

func TestGopassMixedStore(t *testing.T) {
	test.InitTestDirs()

	storeDir := filepath.Join(test.TestData, "gopass-mixed-store")
	_ = os.RemoveAll(storeDir)
	require.NoError(t, os.MkdirAll(storeDir, 0700))

	gpgPub, gpgPriv := setupGopassGPGKeys(t)

	agePubFile := filepath.Join(test.TestData, "gopass_mixed_age"+pubAgeExt)
	agePrivFile := filepath.Join(test.TestData, "gopass_mixed_age"+privAgeExt)
	_ = os.Remove(agePubFile)
	_ = os.Remove(agePrivFile)
	ageIdentity, _, err := CreateAgeIdentity()
	require.NoError(t, err)
	err = ExportAgeKeyPair(ageIdentity, agePubFile, agePrivFile)
	require.NoError(t, err)

	// write one GPG and one age secret into the same store
	require.NoError(t, GopassWrite(storeDir, "gpg/secret", "gpgpassword", gpgPub, GopassCryptoGPG))
	require.NoError(t, GopassWrite(storeDir, "age/secret", "agepassword", agePubFile, GopassCryptoAge))

	t.Run("GopassList GPG only", func(t *testing.T) {
		secrets, listErr := GopassList(storeDir, GopassCryptoGPG)
		assert.NoError(t, listErr)
		assert.Equal(t, []string{"gpg/secret"}, secrets)
	})

	t.Run("GopassList age only", func(t *testing.T) {
		secrets, listErr := GopassList(storeDir, GopassCryptoAge)
		assert.NoError(t, listErr)
		assert.Equal(t, []string{"age/secret"}, secrets)
	})

	t.Run("GopassList both types", func(t *testing.T) {
		secrets, listErr := GopassList(storeDir, "")
		assert.NoError(t, listErr)
		assert.Len(t, secrets, 2)
		assert.Contains(t, secrets, "gpg/secret")
		assert.Contains(t, secrets, "age/secret")
	})

	t.Run("GopassRead GPG secret", func(t *testing.T) {
		secret, readErr := GopassRead(storeDir, "gpg/secret", gpgPriv, testGPGPass, GopassCryptoGPG)
		assert.NoError(t, readErr)
		assert.Equal(t, "gpgpassword", secret)
	})

	t.Run("GopassRead age secret", func(t *testing.T) {
		secret, readErr := GopassRead(storeDir, "age/secret", agePrivFile, "", GopassCryptoAge)
		assert.NoError(t, readErr)
		assert.Equal(t, "agepassword", secret)
	})

	// verify that requesting the wrong type finds no file
	t.Run("GopassRead GPG secret with age type fails", func(t *testing.T) {
		_, readErr := GopassRead(storeDir, "gpg/secret", agePrivFile, "", GopassCryptoAge)
		assert.Error(t, readErr)
	})

	t.Run("GopassRead age secret with GPG type fails", func(t *testing.T) {
		_, readErr := GopassRead(storeDir, "age/secret", gpgPriv, testGPGPass, GopassCryptoGPG)
		assert.Error(t, readErr)
	})

	_ = age.GenerateX25519Identity // ensure age import is used (avoids false unused-import lint warnings)
}

// sampleGopassConfig returns a minimal gopass config (git-config format) that
// maps two store paths, matching the format used by the gopass CLI.
func sampleGopassConfig(rootPath, mountPath string) string {
	return "[mounts]\n\tpath = " + rootPath + "\n[mounts \"work\"]\n\tpath = " + mountPath + "\n"
}

func TestGopassConfigPath(t *testing.T) {
	test.InitTestDirs()

	t.Run("GOPASS_CONFIG env var", func(t *testing.T) {
		_ = os.Setenv(gopassEnvConfig, "/custom/gopass/config")
		p, err := GopassConfigPath()
		assert.NoError(t, err)
		assert.Equal(t, "/custom/gopass/config", p)
		_ = os.Unsetenv(gopassEnvConfig)
	})

	t.Run("XDG_CONFIG_HOME env var", func(t *testing.T) {
		_ = os.Unsetenv(gopassEnvConfig)
		_ = os.Setenv(gopassEnvXDGConfig, "/xdg/config")
		p, err := GopassConfigPath()
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join("/xdg/config", "gopass", "config"), p)
		_ = os.Unsetenv(gopassEnvXDGConfig)
	})

	t.Run("default falls back to ~/.config/gopass/config", func(t *testing.T) {
		_ = os.Unsetenv(gopassEnvConfig)
		_ = os.Unsetenv(gopassEnvXDGConfig)
		_ = os.Unsetenv(gopassEnvHomeDir)
		p, err := GopassConfigPath()
		assert.NoError(t, err)
		assert.Contains(t, filepath.ToSlash(p), ".config/gopass/config")
	})

	t.Run("GOPASS_HOMEDIR overrides home for config path", func(t *testing.T) {
		_ = os.Unsetenv(gopassEnvConfig)
		_ = os.Unsetenv(gopassEnvXDGConfig)
		_ = os.Setenv(gopassEnvHomeDir, "/custom/home")
		p, err := GopassConfigPath()
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join("/custom/home", ".config", "gopass", "config"), p)
		_ = os.Unsetenv(gopassEnvHomeDir)
	})
}

func TestGopassReadConfig(t *testing.T) {
	test.InitTestDirs()
	cfgFile := filepath.Join(test.TestData, "gopass_test_config.yml")

	t.Run("valid config with root and mount", func(t *testing.T) {
		content := sampleGopassConfig("/stores/root", "/stores/work")
		require.NoError(t, common.WriteStringToFile(cfgFile, content))
		cfg, err := GopassReadConfig(cfgFile)
		assert.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "/stores/root", cfg.Root.Path)
		require.Contains(t, cfg.Mounts, "work")
		assert.Equal(t, "/stores/work", cfg.Mounts["work"].Path)
	})

	t.Run("config with only root path", func(t *testing.T) {
		require.NoError(t, common.WriteStringToFile(cfgFile, "[mounts]\n\tpath = /stores/root\n"))
		cfg, err := GopassReadConfig(cfgFile)
		assert.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "/stores/root", cfg.Root.Path)
		assert.Empty(t, cfg.Mounts)
	})

	t.Run("missing config file returns error", func(t *testing.T) {
		cfg, err := GopassReadConfig(filepath.Join(test.TestData, "nonexistent_gopass_config.yml"))
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("invalid or empty config returns empty result without error", func(t *testing.T) {
		require.NoError(t, common.WriteStringToFile(cfgFile, "not a valid config at all\n"))
		cfg, err := GopassReadConfig(cfgFile)
		assert.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Empty(t, cfg.Root.Path)
		assert.Empty(t, cfg.Mounts)
	})
}

func TestGopassDetectCrypto(t *testing.T) {
	test.InitTestDirs()

	// helper: create a fresh empty store dir
	newStore := func(name string) string {
		dir := filepath.Join(test.TestData, name)
		_ = os.RemoveAll(dir)
		require.NoError(t, os.MkdirAll(dir, 0700))
		return dir
	}

	t.Run("age marker file (.age-recipients)", func(t *testing.T) {
		dir := newStore("detect-age-marker")
		require.NoError(t, common.WriteStringToFile(filepath.Join(dir, gopassAgeMarker), "age1abc\n"))
		ct, err := GopassDetectCrypto(dir)
		assert.NoError(t, err)
		assert.Equal(t, GopassCryptoAge, ct)
	})

	t.Run("GPG marker file (.gpg-id) takes precedence over nothing", func(t *testing.T) {
		dir := newStore("detect-gpg-marker")
		require.NoError(t, common.WriteStringToFile(filepath.Join(dir, gopassGPGMarker), "ABC123\n"))
		ct, err := GopassDetectCrypto(dir)
		assert.NoError(t, err)
		assert.Equal(t, GopassCryptoGPG, ct)
	})

	t.Run("age marker takes precedence over GPG marker", func(t *testing.T) {
		dir := newStore("detect-both-markers")
		require.NoError(t, common.WriteStringToFile(filepath.Join(dir, gopassAgeMarker), "age1abc\n"))
		require.NoError(t, common.WriteStringToFile(filepath.Join(dir, gopassGPGMarker), "ABC123\n"))
		ct, err := GopassDetectCrypto(dir)
		assert.NoError(t, err)
		assert.Equal(t, GopassCryptoAge, ct)
	})

	t.Run("extension scan finds .gpg files when no marker", func(t *testing.T) {
		dir := newStore("detect-ext-gpg")
		subDir := filepath.Join(dir, "group")
		require.NoError(t, os.MkdirAll(subDir, 0700))
		require.NoError(t, common.WriteStringToFile(filepath.Join(subDir, "secret.gpg"), "dummy"))
		ct, err := GopassDetectCrypto(dir)
		assert.NoError(t, err)
		assert.Equal(t, GopassCryptoGPG, ct)
	})

	t.Run("extension scan finds .age files when no marker", func(t *testing.T) {
		dir := newStore("detect-ext-age")
		subDir := filepath.Join(dir, "group")
		require.NoError(t, os.MkdirAll(subDir, 0700))
		require.NoError(t, common.WriteStringToFile(filepath.Join(subDir, "secret.age"), "dummy"))
		ct, err := GopassDetectCrypto(dir)
		assert.NoError(t, err)
		assert.Equal(t, GopassCryptoAge, ct)
	})

	t.Run("config lookup finds root store and defaults to GPG", func(t *testing.T) {
		dir := newStore("detect-cfg-root")
		cfgFile := filepath.Join(test.TestData, "gopass_detect_cfg.yml")
		require.NoError(t, common.WriteStringToFile(cfgFile,
			"[mounts]\n\tpath = "+dir+"\n"))
		_ = os.Setenv(gopassEnvConfig, cfgFile)
		ct, err := GopassDetectCrypto(dir)
		_ = os.Unsetenv(gopassEnvConfig)
		assert.NoError(t, err)
		assert.Equal(t, GopassCryptoGPG, ct)
	})

	t.Run("config lookup finds named mount and defaults to GPG", func(t *testing.T) {
		dir := newStore("detect-cfg-mount")
		cfgFile := filepath.Join(test.TestData, "gopass_detect_cfg.yml")
		require.NoError(t, common.WriteStringToFile(cfgFile,
			"[mounts]\n\tpath = /other/store\n[mounts \"work\"]\n\tpath = "+dir+"\n"))
		_ = os.Setenv(gopassEnvConfig, cfgFile)
		ct, err := GopassDetectCrypto(dir)
		_ = os.Unsetenv(gopassEnvConfig)
		assert.NoError(t, err)
		assert.Equal(t, GopassCryptoGPG, ct)
	})

	t.Run("empty store with no config returns error", func(t *testing.T) {
		dir := newStore("detect-empty")
		_ = os.Setenv(gopassEnvConfig, filepath.Join(test.TestData, "no_such_config.yml"))
		ct, err := GopassDetectCrypto(dir)
		_ = os.Unsetenv(gopassEnvConfig)
		assert.Error(t, err)
		assert.Empty(t, ct)
	})

	t.Run("nonexistent store returns error", func(t *testing.T) {
		ct, err := GopassDetectCrypto(filepath.Join(test.TestData, "no-such-store"))
		assert.Error(t, err)
		assert.Empty(t, ct)
	})
}
