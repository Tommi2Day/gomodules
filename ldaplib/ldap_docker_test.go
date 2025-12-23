package ldaplib

import (
	"fmt"
	"os"
	"time"

	"github.com/tommi2day/gomodules/test"

	"github.com/go-ldap/ldap/v3"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/tommi2day/gomodules/common"
)

const Ldaprepo = "docker.io/cleanstart/openldap"
const LdaprepoTag = "2.6.10"
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
	if err != nil || pool == nil {
		return
	}
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + Ldaprepo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, LdaprepoTag)
	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        LdaprepoTag,

		Mounts: []string{
			test.TestDir + "/docker/ldap/certs:/certs:ro",
			// test.TestDir + "/docker/ldap/schema:/schema:ro",
			test.TestDir + "/docker/ldap/ldif:/ldif:ro",
			test.TestDir + "/docker/ldap/etc/slapd.conf:/etc/openldap/slapd.conf:ro",
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
	myhost, myport := common.GetContainerHostAndPort(container, "389/tcp")
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
