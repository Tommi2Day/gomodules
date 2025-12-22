package dblib

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/netlib"
	"github.com/tommi2day/gomodules/test"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	dblibDNSContainerTimeout = 10
	dblibNetworkName         = "dblib-dns"
	dblibNetworkPrefix       = "172.24.0"
	dblibRepoTag             = "9.20"
	dblibDNSPort             = 9054
	dblibTestAddr            = racaddr
)

var (
	dblibDNSContainerName  string
	dblibDNSContainer      *dockertest.Resource
	dblibDNSNetwork        *dockertest.Network
	dblibDNSNetworkCreated = false
	dblibDNSServer         = "127.0.0.1"
)

// prepareDBlibDNSContainer create a Bind9 Docker Container
func prepareDBlibDNSContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_DB_DNS") != "" {
		return nil, fmt.Errorf("skipping DB DNS Container in CI environment")
	}

	dblibDNSContainerName = getContainerName()
	pool, err := common.GetDockerPool()
	if err != nil {
		return nil, err
	}

	err = setupNetwork(pool)
	if err != nil {
		return nil, err
	}

	container, err = buildAndRunContainer(pool)
	if err != nil {
		return
	}

	time.Sleep(10 * time.Second)

	err = validateContainerIP(container)
	if err != nil {
		return
	}

	err = waitForDNSServer(pool)
	if err != nil {
		return
	}

	err = testDNSResolution()
	return
}

func getContainerName() string {
	name := os.Getenv("DBDNS_CONTAINER_NAME")
	if name == "" {
		name = "dblib-bind9"
	}
	return name
}

func setupNetwork(pool *dockertest.Pool) error {
	networks, err := pool.NetworksByName(dblibNetworkName)
	if err != nil || len(networks) == 0 {
		return createNetwork(pool)
	}
	dblibDNSNetwork = &networks[0]
	return nil
}

func createNetwork(pool *dockertest.Pool) error {
	var err error
	dblibDNSNetwork, err = pool.CreateNetwork(dblibNetworkName, func(options *docker.CreateNetworkOptions) {
		options.Name = dblibNetworkName
		options.CheckDuplicate = true
		options.IPAM = &docker.IPAMOptions{
			Driver: "default",
			Config: []docker.IPAMConfig{{
				Subnet:  dblibNetworkPrefix + ".0/24",
				Gateway: dblibNetworkPrefix + ".1",
			}},
		}
		options.EnableIPv6 = false
	})
	if err != nil {
		return fmt.Errorf("could not create Network: %s:%s", dblibNetworkName, err)
	}
	dblibDNSNetworkCreated = true
	return nil
}

func buildAndRunContainer(pool *dockertest.Pool) (*dockertest.Resource, error) {
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	fmt.Printf("Try to build and start docker container %s\n", dblibDNSContainerName)
	buildArgs := []docker.BuildArg{
		{Name: "VENDOR_IMAGE_PREFIX", Value: vendorImagePrefix},
		{Name: "BIND9_VERSION", Value: dblibRepoTag},
	}

	dockerContextDir := test.TestDir + "/docker/oracle-dns"
	return pool.BuildAndRunWithBuildOptions(
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: dockerContextDir,
			Dockerfile: "Dockerfile",
		},
		&dockertest.RunOptions{
			Hostname:     dblibDNSContainerName,
			Name:         dblibDNSContainerName,
			Networks:     []*dockertest.Network{dblibDNSNetwork},
			ExposedPorts: []string{"9054/tcp"},
			// need fixed mapping here
			PortBindings: map[docker.Port][]docker.PortBinding{
				"9054/tcp": {
					{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", dblibDNSPort)},
				},
			},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = false
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})
}

func validateContainerIP(container *dockertest.Resource) error {
	ip := container.GetIPInNetwork(dblibDNSNetwork)

	fmt.Printf("DB DNS Container IP: %s\n", ip)
	return nil
}

// func waitForDNSServer(pool *dockertest.Pool, container *dockertest.Resource) error {
func waitForDNSServer(pool *dockertest.Pool) error {
	pool.MaxWait = dblibDNSContainerTimeout * time.Second
	start := time.Now()
	err := pool.Retry(func() error {
		c, err := net.Dial("udp", net.JoinHostPort(dblibDNSContainerName, fmt.Sprintf("%d", dblibDNSPort)))
		if err != nil {
			fmt.Printf("Err:%s\n", err)
			return err
		}
		_ = c.Close()
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not connect to DB DNS Container: %v", err)
	}

	time.Sleep(10 * time.Second)
	elapsed := time.Since(start)
	fmt.Println("DB DNS Container is ready after ", elapsed.Round(time.Millisecond))
	return nil
}

func testDNSResolution() error {
	dns := netlib.NewResolver(dblibDNSServer, dblibDNSPort, true)
	dns.IPv4Only = true
	s := "/udp"
	if dns.TCP {
		s = "/tcp"
	}
	fmt.Printf("resolve on %s:%d%s\n", dns.Nameserver, dns.Port, s)
	ips, err := dns.LookupIP(dblibTestAddr)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("could not resolve DNS for %s: %v", dblibTestAddr, err)
	}
	fmt.Printf("Host %s resolved to %s\n", dblibTestAddr, ips[0])
	return nil
}

func destroyDNSContainer(container *dockertest.Resource) {
	if container != nil {
		common.DestroyDockerContainer(container)
	}

	if dblibDNSNetworkCreated && dblibDNSNetwork != nil {
		_ = dblibDNSNetwork.Close()
	}
}
