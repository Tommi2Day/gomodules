package pwlib

import (
	"context"
	"fmt"
	"path"
	"strings"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

// VaultData is the data structure
type VaultData map[string]interface{}

// VaultConfig create a new vault client
func VaultConfig(address string, token string) (client *vault.Client, err error) {
	config := vault.DefaultConfig()
	if address != "" {
		config.Address = address
	}
	client, err = vault.NewClient(config)
	if err != nil {
		err = fmt.Errorf("vault client for %s failed:%s", address, err)
		return
	}
	if client == nil {
		err = fmt.Errorf("vault client for %s not created", address)
		return
	}
	if token != "" {
		log.Debugf("set token to %s", token)
		client.SetToken(token)
	}
	log.Debugf("vault client for %s created", address)
	return
}

// VaultKVRead read a KVv2 secret from given mount and path
func VaultKVRead(client *vault.Client, mount string, secretPath string) (vaultSecret *vault.KVSecret, err error) {
	vaultSecret, err = client.KVv2(mount).Get(context.Background(), secretPath)
	if err != nil {
		err = fmt.Errorf("read vault secret failed:%s", err)
		return
	}
	log.Debugf("got secret on path %s ", secretPath)
	return
}

// VaultKVWrite write a KVv2 secret to given mount and path
func VaultKVWrite(client *vault.Client, mount string, secretPath string, data map[string]interface{}) (err error) {
	_, err = client.KVv2(mount).Put(context.Background(), secretPath, data)
	if err != nil {
		err = fmt.Errorf("write vault secret to %s failed:%s", secretPath, err)
		return
	}
	log.Debugf("write secret to path %s successfully", secretPath)
	return
}

// VaultRead logical read path value
func VaultRead(client *vault.Client, path string) (vaultSecret *vault.Secret, err error) {
	vaultSecret, err = client.Logical().Read(path)
	if err != nil {
		err = fmt.Errorf("read vault secret failed:%s", err)
		return
	}
	log.Debugf("read on path %s OK", path)
	return
}

// VaultList recursively lists all secret paths under the given vaultPath using logical metadata listing.
// It returns a flat list of all secret paths found under the specified vaultPath.
func VaultList(client *vault.Client, vaultSecretMount string, vaultPath string) (entries []string, err error) {
	metaPath := vaultSecretMount + "/metadata/"
	if !strings.HasPrefix(vaultPath, metaPath) {
		vaultPath = path.Join(metaPath, vaultPath)
	}
	vaultSecret, err := client.Logical().List(vaultPath)
	if err != nil {
		err = fmt.Errorf("read vault secret failed:%s", err)
		return
	}
	if vaultSecret == nil || vaultSecret.Data == nil {
		return
	}
	keys, ok := vaultSecret.Data["keys"].([]interface{})
	if !ok {
		return
	}
	for _, key := range keys {
		k, ok := key.(string)
		if !ok {
			continue
		}
		if strings.HasSuffix(k, "/") {
			subPath := vaultPath
			if !strings.HasSuffix(subPath, "/") {
				subPath += "/"
			}
			childEntries, childErr := VaultList(client, vaultSecretMount, subPath+k)
			if childErr != nil {
				log.Warnf("error listing subpath %s: %v", subPath+k, childErr)
				continue
			}
			entries = append(entries, childEntries...)
		} else {
			fullPath := vaultPath
			if !strings.HasSuffix(fullPath, "/") {
				fullPath += "/"
			}
			entries = append(entries, fullPath+k)
		}
	}
	// replace metadata paths with data paths
	/*
		for i, entry := range entries {
			entries[i] = strings.Replace(entry, metaPath+"/", "", 1)
		}
	*/
	log.Debugf("recursively listed paths under %s", path.Join(metaPath, vaultPath))
	return
}

// VaultWrite write a value to the given path
func VaultWrite(client *vault.Client, path string, data map[string]interface{}) (err error) {
	_, err = client.Logical().Write(path, data)
	if err != nil {
		err = fmt.Errorf("write to %s failed:%s", path, err)
		return
	}
	log.Debugf("write to path %s successfully", path)
	return
}

// GetVaultSecret reads a vault path as system via logical method and returns secret keys and values as plaintext format
func GetVaultSecret(vaultPath string, vaultAddr string, vaultToken string) (content string, err error) {
	var vc *vault.Client
	var vs *vault.Secret
	var vaultdata map[string]interface{}
	log.Debugf("Vault Read entered for path '%s'", vaultPath)
	vc, _ = VaultConfig(vaultAddr, vaultToken)
	vs, err = VaultRead(vc, vaultPath)
	if err == nil {
		sysKey := strings.ReplaceAll(vaultPath, ":", "_")
		if vs != nil {
			if vd, ok := vs.Data["data"]; ok {
				vaultdata = vd.(map[string]interface{})
				log.Debug("Vault Read OK")
				for k, v := range vaultdata {
					content += fmt.Sprintf("%s:%s:%v\n", sysKey, k, v.(string))
				}
			} else {
				err = fmt.Errorf("no vault data returned")
			}
		} else {
			err = fmt.Errorf("no entries returned")
		}
	}
	return
}
