package pwlib

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tommi2day/gomodules/test"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const repo = "docker.io/hashicorp/vault"
const repoTag = "1.13.1"
const containerTimeout = 120
const rootToken = "pwlib-test"

var containerName string
var pool *dockertest.Pool

// prepareVaultContainer create an Oracle Docker Container
func prepareVaultContainer() (container *dockertest.Resource, err error) {
	pool = nil
	if os.Getenv("SKIP_VAULT") != "" {
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
			"VAULT_DEV_ROOT_TOKEN_ID=" + rootToken,
			"VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200",
		},
		Hostname: containerName,
		Name:     containerName,
		CapAdd:   []string{"IPC_LOCK"},
		Cmd:      []string{},
		// ExposedPorts: []string{"8200"},
		/*
			PortBindings: map[docker.Port][]docker.PortBinding{
				"8200": {
					{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", port)},
				},
			},
		*/
		Mounts: []string{
			test.TestDir + "/vault_provision:/vault_provision/",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = fmt.Errorf("error starting vault docker container: %v", err)
		return
	}

	pool.MaxWait = containerTimeout * time.Second
	host, port := getHostAndPort(container, "8200/tcp")
	address := fmt.Sprintf("http://%s:%d", host, port)
	fmt.Printf("Wait to successfully connect to Vault with %s (max %ds)...\n", address, containerTimeout)
	start := time.Now()
	if err = pool.Retry(func() error {
		var resp *http.Response
		//nolint gosec
		resp, err = http.Get(address)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code not OK:%s", resp.Status)
		}
		return nil
	}); err != nil {
		fmt.Printf("Could not connect to Vault Container: %s", err)
		return
	}

	// wait 5s to init container
	time.Sleep(5 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("vault Container is available after %s\n", elapsed.Round(time.Millisecond))

	// provision
	execCmd(container, []string{"/vault_provision/vault_init.sh"})
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

// executes an OS cmd within container and print output
func execCmd(container *dockertest.Resource, cmd []string) {
	var cmdout bytes.Buffer
	cmdout.Reset()
	_, err := container.Exec(cmd, dockertest.ExecOptions{StdOut: &cmdout})
	if err != nil {
		fmt.Printf("Exec Error %s", err)
	} else {
		fmt.Printf("Cmd:%v\n %s", cmd, cmdout.String())
	}
}
