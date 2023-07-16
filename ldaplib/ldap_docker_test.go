package ldaplib

import (
	"fmt"
	"os"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/tommi2day/gomodules/common"
)

const repo = "docker.io/osixia/openldap"
const repoTag = "1.5.0"
const containerTimeout = 120

var containerName string
var ldapContainer *dockertest.Resource

// prepareContainer create an Oracle Docker Container
func prepareContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_LDAP") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}
	var pool *dockertest.Pool
	containerName = os.Getenv("CONTAINER_NAME")
	pool, err = common.GetDockerPool()
	if err != nil {
		err = fmt.Errorf("could not connect to Docker: %s", err)
		return
	}

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + repo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, repoTag)
	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        repoTag,
		Env: []string{
			"LDAP_ORGANISATION=" + ldapOrganisation,
			"LDAP_DOMAIN=" + LdapDomain,
			"LDAP_BASE_DN=" + LdapBaseDn,
			"LDAP_ADMIN_PASSWORD=" + LdapAdminPassword,
			"LDAP_CONFIG_PASSWORD=" + LdapConfigPassword,
			"LDAP_TLS_VERIFY_CLIENT=never",
		},
		Hostname: containerName,
		Name:     containerName,
		// ExposedPorts: []string{"389", "636"},
		/*
			PortBindings: map[docker.Port][]docker.PortBinding{
				"389": {
					{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", ldapPort)},
				},
				"636": {
					{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", sslport)},
				},
			},
		*/
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = fmt.Errorf("error starting ldap docker container: %v", err)
		return
	}

	pool.MaxWait = containerTimeout * time.Second
	host, port := common.GetContainerHostAndPort(container, "389/tcp")
	dialURL := fmt.Sprintf("ldap://%s:%d", host, port)
	fmt.Printf("Wait to successfully connect to Ldap with %s (max %ds)...\n", dialURL, containerTimeout)
	start := time.Now()
	var l *ldap.Conn
	if err = pool.Retry(func() error {
		l, err = ldap.DialURL(dialURL)
		return err
	}); err != nil {
		fmt.Printf("Could not connect to LDAP Container: %s", err)
		return
	}
	_ = l.Close()
	// wait 5s to init container
	time.Sleep(5 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("LDAP Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}
