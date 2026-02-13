package pwlib

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/tommi2day/gomodules/common"

	"github.com/tommi2day/gomodules/test"

	vault "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const vaultTest1 = "test"
const vaultTest2 = "logical/vaultTest2"
const vaultTest3 = "logical/dir/vaultTest3"
const vaultSecretMount = "secret"

func TestVault(t *testing.T) {
	var vc *vault.Client
	var kvs *vault.KVSecret
	var vs *vault.Secret
	if os.Getenv("SKIP_Vault") != "" {
		t.Skip("Skipping Vault testing in CI environment")
	}
	vaultContainer, err := prepareVaultContainer()
	defer common.DestroyDockerContainer(vaultContainer)
	require.NoErrorf(t, err, "Vault Server not available")
	require.NotNil(t, vaultContainer, "Prepare failed")
	if err != nil || vaultContainer == nil {
		t.Fatal("Vault Server not available")
	}
	host, vaultPort := common.GetContainerHostAndPort(vaultContainer, "8200/tcp")
	address := fmt.Sprintf("http://%s:%d", host, vaultPort)
	_ = os.Unsetenv("VAULT_ADDR")
	_ = os.Unsetenv("VAULT_TOKEN")
	t.Run("Vault Connect direct", func(t *testing.T) {
		t.Logf("Connect to vault using %s and token %s", address, rootToken)
		vc, err = VaultConfig(address, rootToken)
		// validate connect with lookup myself
		secret, err := vc.Auth().Token().LookupSelf()
		require.NoErrorf(t, err, "Token Lookup returned error: %v", err)
		assert.NotNilf(t, secret, "Vault Token is nil")
		vc.ClearToken()
	})
	t.Run("Vault Wrong Token", func(t *testing.T) {
		t.Logf("Connect to vault using %s and token %s", address, "xxx")
		vc, err = VaultConfig(address, "xxx")
		// validate connect with lookup myself
		secret, err := vc.Auth().Token().LookupSelf()
		require.Error(t, err, "Test should fail")
		assert.Nilf(t, secret, "Vault auth should not return")
		vc.ClearToken()
	})
	t.Run("Vault Connect with Env", func(t *testing.T) {
		t.Log("Connect to vault using env")
		_ = os.Setenv("VAULT_ADDR", address)
		_ = os.Setenv("VAULT_TOKEN", rootToken)
		vc, err = VaultConfig("", "")
		// validate connect with lookup myself
		secret, err := vc.Auth().Token().LookupSelf()
		require.NoErrorf(t, err, "Token Lookup returned error: %v", err)
		assert.NotNilf(t, secret, "Vault auth should not return")
		vc.ClearToken()
		_ = os.Unsetenv("VAULT_ADDR")
		_ = os.Unsetenv("VAULT_TOKEN")
	})

	vc, err = VaultConfig(address, rootToken)
	// validate connect with lookup myself
	secret, err := vc.Auth().Token().LookupSelf()
	require.NoErrorf(t, err, "Connect returned error: %v", err)
	assert.NotNilf(t, secret, "Vault auth should not return")
	t.Run("Vault KV Write", func(t *testing.T) {
		var vaultdata = map[string]interface{}{
			"password": "Hashi123",
		}
		err = VaultKVWrite(vc, "secret", vaultTest1, vaultdata)
		require.NoErrorf(t, err, "Write returned error: %v", err)
	})
	t.Run("Vault KV Wrong Path", func(t *testing.T) {
		kvs, err = VaultKVRead(vc, "secret", "test-wrong")
		require.Error(t, err, "Read should return an error")
		require.Nilf(t, kvs, "Vault Secret should be nil")
	})
	t.Run("Vault KV Read", func(t *testing.T) {
		kvs, err = VaultKVRead(vc, "secret", vaultTest1)
		require.NoErrorf(t, err, "Read returned error: %v", err)
		require.NotNilf(t, kvs, "Vault Secret is nil")
		value, success := kvs.Data["password"].(string)
		require.True(t, success, "Key password not found")
		assert.Equalf(t, value, "Hashi123", "unexpected password value %q retrieved from vault", value)
	})
	t.Run("Vault logical Write", func(t *testing.T) {
		var vaultdata = map[string]interface{}{
			"data": map[string]interface{}{
				"password": "Hashi345",
			},
		}
		err = VaultWrite(vc, path.Join("secret", "data", vaultTest2), vaultdata)
		require.NoErrorf(t, err, "Write returned error: %v", err)
		err = VaultWrite(vc, path.Join("secret", "data", vaultTest3), vaultdata)
		require.NoErrorf(t, err, "Write returned error: %v", err)
	})
	t.Run("Vault List", func(t *testing.T) {
		var entries []string
		tests := []string{vaultTest1, vaultTest2, vaultTest3}
		entries, err = VaultList(vc, vaultSecretMount, "")
		t.Logf("Vault List returned entries: %v", entries)
		require.NoErrorf(t, err, "List returned error: %v", err)
		// Expecting a flat list of all secret paths
		// Example: ["data/test", "data/logical/vaultTest2"]
		assert.GreaterOrEqual(t, len(entries), len(entries), "Should return %d secret path entries, but got %d", len(tests), len(entries))
		for _, entry := range tests {
			found := false
			for _, e := range entries {
				if strings.HasSuffix(e, entry) {
					found = true
					break
				}
			}
			assert.Truef(t, found, "Expected secret path %s not found", entry)
			t.Logf("secret path: %s", entry)
		}
	})
	t.Run("Vault Logical Read", func(t *testing.T) {
		var vaultdata map[string]interface{}
		vs, err = VaultRead(vc, path.Join("secret", "data", vaultTest2))
		require.NoErrorf(t, err, "Read returned error: %v", err)
		require.NotNilf(t, vs, "Vault Secret is nil")
		vaultwarn := vs.Warnings
		assert.Nil(t, vaultwarn, "Should have no warnings, but got %v", vaultwarn)
		require.NotNil(t, vs.Data, "No Data returned")
		success := false
		value := ""
		vaultdata = vs.Data["data"].(map[string]interface{})
		value, success = vaultdata["password"].(string)
		require.True(t, success, "Key password not found")
		assert.Equalf(t, value, "Hashi345", "unexpected password value %q retrieved from vault", value)
	})
	t.Run("Vault GetPassword", func(t *testing.T) {
		// need Env as config is here not exposed
		_ = os.Setenv("VAULT_ADDR", address)
		_ = os.Setenv("VAULT_TOKEN", rootToken)
		app := "test_get_pass_vault"
		pc := NewConfig(app, test.TestData, test.TestData, app, typeVault)
		pass, err := pc.GetPassword("/secret/data/"+vaultTest2, "password")
		expected := "Hashi345"
		assert.NoErrorf(t, err, "Got unexpected error: %s", err)
		assert.Equal(t, expected, pass, "Answer not expected. exp:%s,act:%s", expected, pass)
	})
	t.Run("Vault GetPassword fail", func(t *testing.T) {
		// need Env as config is here not exposed
		_ = os.Setenv("VAULT_ADDR", address)
		_ = os.Setenv("VAULT_TOKEN", rootToken)
		app := "test_get_pass_vault_fail"
		pc := NewConfig(app, test.TestData, test.TestData, app, typeVault)
		pass, err := pc.GetPassword("/secret/vaultTest2", "password")
		assert.Error(t, err, "Should have failed")
		assert.Emptyf(t, pass, "pass Should be emty, but have %s", pass)
		if err != nil {
			t.Log(err)
		}
	})
}
