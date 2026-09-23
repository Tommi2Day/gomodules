package pwlib

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ory/dockertest/v4"
	"github.com/tommi2day/gomodules/common"

	"github.com/moby/moby/api/types/container"
)

const secretsManagerImage = "docker.io/motoserver/moto" //nolint:gosec // docker image reference, not a credential
const secretsManagerImageTag = "5.2.3"
const secretsManagerContainerTimeout = 120

var secretsManagerContainerName string

// https://github.com/getmoto/moto
// prepareSecretsManagerContainer starts a moto container that mocks the AWS Secrets Manager API
func prepareSecretsManagerContainer() (resource dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_SECRETSMANAGER") != "" {
		err = fmt.Errorf("skipping Secrets Manager Container in CI environment")
		return
	}
	secretsManagerContainerName = os.Getenv("SECRETSMANAGER_CONTAINER_NAME")
	if secretsManagerContainerName == "" {
		secretsManagerContainerName = "pwlib-secretsmanager"
	}
	pool, err := common.GetDockerPool()
	if err != nil || pool == nil {
		err = fmt.Errorf("cannot attach to docker: %v", err)
		return
	}

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + secretsManagerImage

	ctx := context.Background()
	fmt.Printf("Try to start docker container for %s:%s\n", repoString, secretsManagerImageTag)
	resource, err = pool.Run(ctx, repoString,
		dockertest.WithTag(secretsManagerImageTag),
		dockertest.WithHostname(secretsManagerContainerName),
		dockertest.WithName(secretsManagerContainerName),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
		}),
	)

	if err != nil || resource == nil {
		err = fmt.Errorf("error starting Secrets Manager docker container: %v", err)
		return
	}

	host, port := common.GetContainerHostAndPort(resource, "5000/tcp")
	address := fmt.Sprintf("http://%s:%d", host, port)
	fmt.Printf("Wait to successfully connect to Secrets Manager with %s (max %ds)...\n", address, secretsManagerContainerTimeout)
	start := time.Now()
	if err = pool.Retry(ctx, secretsManagerContainerTimeout*time.Second, func() error {
		var resp *http.Response
		//nolint gosec
		resp, err = http.Get(address)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code not OK:%s", resp.Status)
		}
		return nil
	}); err != nil {
		fmt.Printf("Could not connect to Secrets Manager Container: %s", err)
		return
	}

	elapsed := time.Since(start)
	fmt.Printf("Secrets Manager Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}
