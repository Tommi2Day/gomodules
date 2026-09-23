package pwlib

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/ory/dockertest/v4"
	"github.com/tommi2day/gomodules/common"
)

const kmsImage = "docker.io/motoserver/moto"
const kmsImageTag = "5.2.3"
const kmsContainerTimeout = 120
const kmsPort = 18080

var kmsContainerName string
var kmsHost = common.GetEnv("KMS_HOST", "127.0.0.1")
var kmsAddress = fmt.Sprintf("http://%s:%d", kmsHost, kmsPort)

// https://github.com/getmoto/moto
// prepareKmsContainer starts a moto container that mocks the AWS KMS API
func prepareKmsContainer() (kmsResource dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_KMS") != "" {
		err = fmt.Errorf("skipping KMS Container in CI environment")
		return
	}
	kmsContainerName = os.Getenv("KMS_CONTAINER_NAME")
	if kmsContainerName == "" {
		kmsContainerName = "pwlib-kms"
	}
	pool, err := common.GetDockerPool()
	if err != nil || pool == nil {
		err = fmt.Errorf("cannot attach to docker: %v", err)
		return
	}

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + kmsImage

	ctx := context.Background()
	fmt.Printf("Try to start docker kmsContainer for %s:%s\n", kmsImage, kmsImageTag)
	kmsResource, err = pool.Run(ctx, repoString,
		dockertest.WithTag(kmsImageTag),
		dockertest.WithHostname(kmsContainerName),
		dockertest.WithName(kmsContainerName),
		dockertest.WithContainerConfig(func(config *container.Config) {
			if config.ExposedPorts == nil {
				config.ExposedPorts = network.PortSet{}
			}
			config.ExposedPorts[network.MustParsePort("5000/tcp")] = struct{}{}
		}),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			// set AutoRemove to true so that stopped kmsContainer goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
			config.PortBindings = network.PortMap{
				network.MustParsePort("5000/tcp"): {
					{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: fmt.Sprintf("%d", kmsPort)},
				},
			}
		}),
	)

	if err != nil {
		err = fmt.Errorf("error starting KMS docker kmsContainer: %v", err)
		return
	}

	fmt.Printf("Wait to successfully connect to KMS with %s (max %ds)...\n", kmsAddress, kmsContainerTimeout)
	start := time.Now()
	if err = pool.Retry(ctx, kmsContainerTimeout*time.Second, func() error {
		var resp *http.Response
		//nolint gosec
		resp, err = http.Get(kmsAddress)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code not OK:%s", resp.Status)
		}
		return nil
	}); err != nil {
		fmt.Printf("Could not connect to KMS Container: %s", err)
		return
	}

	elapsed := time.Since(start)
	fmt.Printf("KMS Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}
