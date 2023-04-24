package pwlib

import (
	"context"
	"fmt"

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

// VaultList logical list path
func VaultList(client *vault.Client, path string) (vaultSecret *vault.Secret, err error) {
	vaultSecret, err = client.Logical().List(path)
	if err != nil {
		err = fmt.Errorf("read vault secret failed:%s", err)
		return
	}
	log.Debugf("read on path %s OK", path)
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
