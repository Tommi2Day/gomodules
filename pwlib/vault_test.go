package pwlib

import (
	"fmt"
	"os"
	"path"
	"testing"

	vault "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVault(t *testing.T) {
	var vc *vault.Client
	var kvs *vault.KVSecret
	var vs *vault.Secret
	if os.Getenv("SKIP_Vault") != "" {
		t.Skip("Skipping Vault testing in CI environment")
	}
	vaultContainer, err := prepareVaultContainer()
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, vaultContainer, "Prepare failed")
	defer destroyContainer(vaultContainer)

	host, port := getHostAndPort(vaultContainer, "8200/tcp")
	address := fmt.Sprintf("http://%s:%d", host, port)
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
		err = VaultKVWrite(vc, "secret", "test", vaultdata)
		require.NoErrorf(t, err, "Write returned error: %v", err)
	})
	t.Run("Vault KV Wrong Path", func(t *testing.T) {
		kvs, err = VaultKVRead(vc, "secret", "test-wrong")
		require.Error(t, err, "Read should return an error")
		require.Nilf(t, kvs, "Vault Secret should be nil")
	})
	t.Run("Vault KV Read", func(t *testing.T) {
		kvs, err = VaultKVRead(vc, "secret", "test")
		require.NoErrorf(t, err, "Read returned error: %v", err)
		require.NotNilf(t, kvs, "Vault Secret is nil")
		value, ok := kvs.Data["password"].(string)
		require.True(t, ok, "Key password not found")
		assert.Equalf(t, value, "Hashi123", "unexpected password value %q retrieved from vault", value)
	})
	t.Run("Vault logical Write", func(t *testing.T) {
		var vaultdata = map[string]interface{}{
			"data": map[string]interface{}{
				"password": "Hashi345",
			},
		}
		err = VaultWrite(vc, path.Join("secret", "data", "test2"), vaultdata)
		require.NoErrorf(t, err, "Write returned error: %v", err)
	})
	t.Run("Vault List", func(t *testing.T) {
		var vaultkeys []interface{}
		vs, err = VaultList(vc, path.Join("secret", "metadata"))
		require.NoErrorf(t, err, "List returned error: %v", err)
		require.NotNilf(t, vs, "Vault Secret is nil")
		vaultwarn := vs.Warnings
		assert.Nil(t, vaultwarn, "Should have no warnings, but got %v", vaultwarn)
		require.NotNil(t, vs.Data, "No Data returned")

		vaultkeys = vs.Data["keys"].([]interface{})
		require.NotNil(t, vaultkeys, "No Keys returned")
		assert.Equalf(t, 2, len(vaultkeys), "Returned key count not as expected")
		for i, k := range vaultkeys {
			t.Logf("key returned %d:%s", i, k)
		}
	})
	t.Run("Vault Logical Read", func(t *testing.T) {
		var vaultdata map[string]interface{}
		vs, err = VaultRead(vc, path.Join("secret", "data", "test2"))
		require.NoErrorf(t, err, "Read returned error: %v", err)
		require.NotNilf(t, vs, "Vault Secret is nil")
		vaultwarn := vs.Warnings
		assert.Nil(t, vaultwarn, "Should have no warnings, but got %v", vaultwarn)
		require.NotNil(t, vs.Data, "No Data returned")
		ok := false
		value := ""
		vaultdata = vs.Data["data"].(map[string]interface{})
		value, ok = vaultdata["password"].(string)
		require.True(t, ok, "Key password not found")
		assert.Equalf(t, value, "Hashi345", "unexpected password value %q retrieved from vault", value)
	})
}
