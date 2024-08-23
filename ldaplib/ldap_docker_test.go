package ldaplib

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/tommi2day/gomodules/test"

	"github.com/go-ldap/ldap/v3"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/tommi2day/gomodules/common"
)

const Ldaprepo = "docker.io/bitnami/openldap"
const LdaprepoTag = "2.6.8"
const LdapcontainerTimeout = 120

var ldapcontainerName string
var ldapContainer *dockertest.Resource

// prepareContainer create an OpenLdap Docker Container
func prepareLdapContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_LDAP") != "" {
		err = fmt.Errorf("skipping LDAP Container in CI environment")
		return
	}
	ldapcontainerName = os.Getenv("LDAP_CONTAINER_NAME")
	if ldapcontainerName == "" {
		ldapcontainerName = "ldaplib-openldap"
	}

	var pool *dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + Ldaprepo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, LdaprepoTag)
	fmt.Println(path.Join(test.TestDir, "docker", "ldap", "certs") + ":/opt/bitnami/openldap/certs:ro")
	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        LdaprepoTag,
		Env: []string{

			"LDAP_PORT_NUMBER=1389",
			"LDAP_LDAPS_PORT_NUMBER=1636",
			"BITNAMI_DEBUG=true",
			"LDAP_ROOT=" + LdapBaseDn,
			"LDAP_ADMIN_USERNAME=admin",
			"LDAP_ADMIN_PASSWORD=" + LdapAdminPassword,
			"LDAP_CONFIG_ADMIN_ENABLED=yes",
			"LDAP_CONFIG_ADMIN_USERNAME=config",
			"LDAP_CONFIG_ADMIN_PASSWORD=" + LdapConfigPassword,
			// "LDAP_SKIP_DEFAULT_TREE=yes",
			// "LDAP_CUSTOM_LDIF_DIR=/bootstrap/ldif",
			"LDAP_CUSTOM_SCHEMA_DIR=/bootstrap/schema",
			"LDAP_ADD_SCHEMAS=yes",
			"LDAP_EXTRA_SCHEMAS=cosine,inetorgperson,nis",
			"LDAP_ALLOW_ANON_BINDING=yes",
			"LDAP_ENABLE_TLS=yes",
			"LDAP_TLS_CERT_FILE=/opt/bitnami/openldap/certs/ldap.example.local-full.crt",
			"LDAP_TLS_KEY_FILE=/opt/bitnami/openldap/certs/ldap.example.local.key",
			"LDAP_TLS_CA_FILE=/opt/bitnami/openldap/certs/ca.crt",
			"LDAP_TLS_VERIFY_CLIENTS=never",
		},

		Mounts: []string{
			test.TestDir + "/docker/ldap/ldif:/bootstrap/ldif:ro",
			test.TestDir + "/docker/ldap/schema:/bootstrap/schema:ro",
			test.TestDir + "/docker/ldap/entrypoint:/docker-entrypoint-initdb.d",
			test.TestDir + "/docker/ldap/certs:/opt/bitnami/openldap/certs:ro",
		},

		Hostname: ldapcontainerName,
		Name:     ldapcontainerName,
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = fmt.Errorf("error starting ldap docker container: %v", err)
		return
	}

	pool.MaxWait = LdapcontainerTimeout * time.Second
	myhost, myport := common.GetContainerHostAndPort(container, "1389/tcp")
	dialURL := fmt.Sprintf("ldap://%s:%d", myhost, myport)
	fmt.Printf("Wait to successfully connect to Ldap with %s (max %ds)...\n", dialURL, LdapcontainerTimeout)
	start := time.Now()
	var l *ldap.Conn
	if err = pool.Retry(func() error {
		l, err = ldap.DialURL(dialURL)
		return err
	}); err != nil {
		fmt.Printf("Could not connect to LDAP Container: %s", err)
		return
	}
	/*
		if err = pool.Retry(func() error {
			err = l.Bind("cn=admin,"+LdapBaseDn, LdapAdminPassword)
			return err
		}); err != nil {
			fmt.Printf("Could not login to LDAP Container: %s", err)
			return
		}
	*/
	_ = l.Close()
	elapsed := time.Since(start)
	fmt.Printf("LDAP Container is available after %s\n", elapsed.Round(time.Millisecond))
	// wait 15s to init container
	time.Sleep(15 * time.Second)
	err = nil
	return
}
