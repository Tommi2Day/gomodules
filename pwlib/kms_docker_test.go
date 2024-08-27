package pwlib

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
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
func prepareKmsContainer() (kmsContainer *dockertest.Resource, err error) {
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

	fmt.Printf("Try to start docker kmsContainer for %s:%s\n", kmsImage, kmsImageTag)
	kmsContainer, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        kmsImageTag,
		Env: []string{
			"PORT=8080",
			"KMS_ACCOUNT_ID=111122223333",
			"KMS_REGION=eu-central-1",
			"KMS_SEED_PATH=/init/seed.yaml",
			"KMS_DATA_PATH=/data",
		},
		Hostname:     kmsContainerName,
		Name:         kmsContainerName,
		ExposedPorts: []string{"8080"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"8080": {
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", kmsPort)},
			},
		},
		Mounts: []string{
			test.TestDir + "/docker/kms/init:/init",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped kmsContainer goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = fmt.Errorf("error starting vault docker kmsContainer: %v", err)
		return
	}

	pool.MaxWait = kmsContainerTimeout * time.Second
	// host, port := common.GetContainerHostAndPort(kmsContainer, "8080/tcp")

	fmt.Printf("Wait to successfully connect to KMS with %s (max %ds)...\n", kmsAddress, kmsContainerTimeout)
	start := time.Now()
	var c net.Conn
	if err = pool.Retry(func() error {
		c, err = net.Dial("tcp", fmt.Sprintf("%s:%d", kmsHost, kmsPort))
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
