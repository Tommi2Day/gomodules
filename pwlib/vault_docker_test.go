package pwlib

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/tommi2day/gomodules/common"

	"github.com/tommi2day/gomodules/test"

	"github.com/moby/moby/api/types/container"
	"github.com/ory/dockertest/v4"
)

const repo = "docker.io/hashicorp/vault"
const repoTag = "1.21.4"
const containerTimeout = 120
const rootToken = "pwlib-test"

var containerName string

// prepareVaultContainer create an Oracle Docker Container
func prepareVaultContainer() (resource dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_VAULT") != "" {
		err = fmt.Errorf("skipping Vault Container in CI environment")
		return
	}
	containerName = os.Getenv("CONTAINER_NAME")
	if containerName == "" {
		containerName = "pwlib-vault"
	}
	pool, err := common.GetDockerPool()
	if err != nil || pool == nil {
		err = fmt.Errorf("cannot attach to docker: %v", err)
		return
	}

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + repo

	ctx := context.Background()
	fmt.Printf("Try to start docker container for %s:%s\n", repoString, repoTag)
	resource, err = pool.Run(ctx, repoString,
		dockertest.WithTag(repoTag),
		dockertest.WithEnv([]string{
			"VAULT_DEV_ROOT_TOKEN_ID=" + rootToken,
			"VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200",
		}),
		dockertest.WithHostname(containerName),
		dockertest.WithName(containerName),
		dockertest.WithCmd([]string{}),
		dockertest.WithMounts([]string{
			test.TestDir + "/docker/vault_provision:/vault_provision/",
		}),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
			config.CapAdd = []string{"IPC_LOCK"}
		}),
	)

	if err != nil || resource == nil {
		err = fmt.Errorf("error starting vault docker container: %v", err)
		return
	}

	host, port := common.GetContainerHostAndPort(resource, "8200/tcp")
	address := fmt.Sprintf("http://%s:%d", host, port)
	fmt.Printf("Wait to successfully connect to Vault with %s (max %ds)...\n", address, containerTimeout)
	start := time.Now()
	if err = pool.Retry(ctx, containerTimeout*time.Second, func() error {
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
	cmdout := ""
	cmd := []string{"bash /vault_provision/vault_init.sh"}
	cmdout, _, err = common.ExecDockerCmd(resource, cmd)
	if err != nil {
		fmt.Printf("Exec Error %s", err)
	} else {
		fmt.Printf("Cmd:%v\n %s", cmd, cmdout)
	}
	err = nil
	return
}
