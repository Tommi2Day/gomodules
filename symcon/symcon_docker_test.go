package symcon

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
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
func prepareIpsContainer() (ipsContainer *dockertest.Resource, err error) {
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

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + ipsImage

	fmt.Printf("Try to start docker ips Container for %s:%s\n", ipsImage, ipsImageTag)
	ipsContainer, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository:   repoString,
		Tag:          ipsImageTag,
		Env:          []string{},
		Hostname:     ipsContainerName,
		Name:         ipsContainerName,
		ExposedPorts: []string{"3777"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"3777": {
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", ipsPort)},
			},
		},
		Mounts: []string{
			test.TestDir + "/docker/symcon:/root",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped kmsContainer goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil || ipsContainer == nil {
		err = fmt.Errorf("error starting docker ips Container: %v", err)
		return
	}

	pool.MaxWait = ipsContainerTimeout * time.Second

	fmt.Printf("Wait to successfully connect to IPS with %s (max %ds)...\n", ipsTestURL, ipsContainerTimeout)
	start := time.Now()

	// wait 15s to init IPS Container
	time.Sleep(15 * time.Second)

	if err = pool.Retry(func() error {
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
