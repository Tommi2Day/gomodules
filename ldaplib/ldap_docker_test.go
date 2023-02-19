package ldaplib

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const repo = "docker.io/osixia/openldap"
const repoTag = "1.5.0"
const containerTimeout = 120

var containerName string
var pool *dockertest.Pool
var ldapContainer *dockertest.Resource

// prepareContainer create an Oracle Docker Container
func prepareContainer() (container *dockertest.Resource, err error) {
	pool = nil
	if os.Getenv("SKIP_LDAP") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}
	containerName = os.Getenv("CONTAINER_NAME")
	pool, err = dockertest.NewPool("")
	if err != nil {
		err = fmt.Errorf("cannot attach to docker: %v", err)
		return
	}
	err = pool.Client.Ping()
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
					{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", port)},
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
	host, port := getHostAndPort(container, "389/tcp")
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
	l.Close()
	// wait 5s to init container
	time.Sleep(5 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("LDAP Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}

func destroyContainer(container *dockertest.Resource) {
	if err := pool.Purge(container); err != nil {
		fmt.Printf("Could not purge resource: %s\n", err)
	}
}

func getHostAndPort(container *dockertest.Resource, portID string) (server string, port int) {
	dockerURL := os.Getenv("DOCKER_HOST")
	if dockerURL == "" {
		address := container.GetHostPort(portID)
		a := strings.Split(address, ":")
		server = a[0]
		port, _ = strconv.Atoi(a[1])
	} else {
		u, _ := url.Parse(dockerURL)
		server = u.Hostname()
		p := container.GetPort(portID)
		port, _ = strconv.Atoi(p)
	}
	return
}
