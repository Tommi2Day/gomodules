package pwlib

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/ory/dockertest/v4"
)

const kmsImage = "docker.io/nsmithuk/local-kms"
const kmsImageTag = "3.12.0"
const kmsContainerTimeout = 120
const kmsPort = 18080

var kmsContainerName string
var kmsHost = common.GetEnv("KMS_HOST", "127.0.0.1")
var kmsAddress = fmt.Sprintf("http://%s:%d", kmsHost, kmsPort)

// https://github.com/nsmithuk/local-kms
// prepareKmsContainer create an Oracle Docker Container
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
		dockertest.WithEnv([]string{
			"PORT=8080",
			"KMS_ACCOUNT_ID=111122223333",
			"KMS_REGION=eu-central-1",
			"KMS_SEED_PATH=/init/seed.yaml",
			"KMS_DATA_PATH=/data",
		}),
		dockertest.WithHostname(kmsContainerName),
		dockertest.WithName(kmsContainerName),
		dockertest.WithMounts([]string{
			test.TestDir + "/docker/kms/init:/init",
		}),
		dockertest.WithContainerConfig(func(config *container.Config) {
			if config.ExposedPorts == nil {
				config.ExposedPorts = network.PortSet{}
			}
			config.ExposedPorts[network.MustParsePort("8080/tcp")] = struct{}{}
		}),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			// set AutoRemove to true so that stopped kmsContainer goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
			config.PortBindings = network.PortMap{
				network.MustParsePort("8080/tcp"): {
					{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: fmt.Sprintf("%d", kmsPort)},
				},
			}
		}),
	)

	if err != nil {
		err = fmt.Errorf("error starting vault docker kmsContainer: %v", err)
		return
	}

	// host, port := common.GetContainerHostAndPort(kmsResource, "8080/tcp")

	fmt.Printf("Wait to successfully connect to KMS with %s (max %ds)...\n", kmsAddress, kmsContainerTimeout)
	start := time.Now()
	var c net.Conn
	if err = pool.Retry(ctx, kmsContainerTimeout*time.Second, func() error {
		c, err = net.Dial("tcp", net.JoinHostPort(kmsHost, fmt.Sprintf("%d", kmsPort)))
		if err != nil {
			fmt.Printf("Err:%s\n", err)
		}
		return err
	}); err != nil {
		fmt.Printf("Could not connect to KMS Container: %d", err)
		return
	}
	_ = c.Close()

	// wait 5s to init kmsContainer
	time.Sleep(5 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("Local KMS Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}
