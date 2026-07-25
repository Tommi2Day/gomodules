package symcon

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
	"github.com/ory/dockertest/v4"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"
)

const ipsImage = "docker.io/symcon/symcon"
const ipsImageTag = "stable"
const ipsContainerTimeout = 120
const ipsPort = 13777
const ipsTestResponse = "IP-Symcon Management Console"

var ipsContainerName string
var ipsHost = common.GetEnv("IPS_HOST", "127.0.0.1")
var ipsURL = fmt.Sprintf("http://%s:%d/api/", ipsHost, ipsPort)
var ipsTestURL = fmt.Sprintf("http://%s:%d/console/", ipsHost, ipsPort)

// https://github.com/nsmithuk/local-kms
// prepareIpsContainer create an Oracle Docker Container
func prepareIpsContainer() (ipsResource dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_IPS") != "" {
		err = fmt.Errorf("skipping IPS Container in CI environment")
		return
	}
	ipsContainerName = os.Getenv("IPS_CONTAINER_NAME")
	if ipsContainerName == "" {
		ipsContainerName = "symconlib-ips"
	}
	pool, err := common.GetDockerPool()
	if err != nil || pool == nil {
		err = fmt.Errorf("cannot attach to docker: %v", err)
		return
	}

	ctx := context.Background()
	// always load image as is the stable tag
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + ipsImage
	pullResp, err := pool.Client().ImagePull(ctx, repoString+":"+ipsImageTag, mobyclient.ImagePullOptions{})
	if err == nil {
		err = pullResp.Wait(ctx)
		_ = pullResp.Close()
	}
	if err != nil {
		err = fmt.Errorf("cannot pull docker image %s:%s (%v)", repoString, ipsImageTag, err)
		return
	}

	fmt.Printf("Try to start docker ips Container for %s:%s\n", ipsImage, ipsImageTag)
	ipsResource, err = pool.Run(ctx, repoString,
		dockertest.WithTag(ipsImageTag),
		dockertest.WithHostname(ipsContainerName),
		dockertest.WithName(ipsContainerName),
		dockertest.WithMounts([]string{
			test.TestDir + "/docker/symcon:/root",
		}),
		dockertest.WithContainerConfig(func(config *container.Config) {
			if config.ExposedPorts == nil {
				config.ExposedPorts = network.PortSet{}
			}
			config.ExposedPorts[network.MustParsePort("3777/tcp")] = struct{}{}
		}),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			// set AutoRemove to true so that stopped kmsContainer goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
			config.PortBindings = network.PortMap{
				network.MustParsePort("3777/tcp"): {
					{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: fmt.Sprintf("%d", ipsPort)},
				},
			}
		}),
	)

	if err != nil || ipsResource == nil {
		err = fmt.Errorf("error starting docker ips Container: %v", err)
		return
	}

	fmt.Printf("Wait to successfully connect to IPS with %s (max %ds)...\n", ipsTestURL, ipsContainerTimeout)
	start := time.Now()

	// wait 15s to init IPS Container
	time.Sleep(15 * time.Second)

	if err = pool.Retry(ctx, ipsContainerTimeout*time.Second, func() error {
		resp, err := common.HTTPGet(ipsTestURL, 2)
		if err != nil {
			return err
		}
		if !strings.Contains(resp, ipsTestResponse) {
			return fmt.Errorf("IPS not ready")
		}
		return nil
	}); err != nil {
		fmt.Printf("Could not connect to IPS: %d", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("Local IPS Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}
