package netlib

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

const (
	netlibDNSContainerTimeout = 10
	netlibNetworkName         = "netlib-dns"
	netlibNetworkPrefix       = "172.25.0"
	netlibDomain              = "netlib.lan"
	netlibTestAddr            = tDB
	netlibRepoTag             = "9.20"
	netlibDNSPort             = 9053
)

var (
	netlibDNSContainerName  string
	netlibDNSContainer      *dockertest.Resource
	netlibDNSNetwork        *dockertest.Network
	netlibDNSNetworkCreated = false
	netlibDNSServer         = "127.0.0.1"
)

// prepareNetlibDNSContainer create a Bind9 Docker Container
func prepareNetlibDNSContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_NET_DNS") != "" {
		return nil, fmt.Errorf("skipping Net DNS Container in CI environment")
	}
	netlibDNSContainerName = getContainerName()
	// use versioned docker pool because of api error client version to old
	pool, err := common.GetVersionedDockerPool("")
	if err != nil {
		return nil, err
	}
	// setup network
	err = setupNetwork(pool)
	if err != nil {
		return nil, err
	}

	container, err = buildAndRunContainer(pool)
	if err != nil {
		return
	}

	time.Sleep(10 * time.Second)

	ip := validateContainerIP(container)
	if ip == "" {
		return
	}

	err = waitForDNSServer(pool, ip)
	if err != nil {
		return
	}

	err = testDNSResolution()
	return
}

func getContainerName() string {
	name := os.Getenv("NETDNS_CONTAINER_NAME")
	if name == "" {
		name = "netlib-bind9"
	}
	return name
}

func setupNetwork(pool *dockertest.Pool) error {
	networks, err := pool.NetworksByName(netlibNetworkName)
	if err != nil || len(networks) == 0 {
		return createNetwork(pool)
	}
	netlibDNSNetwork = &networks[0]
	return nil
}

func createNetwork(pool *dockertest.Pool) error {
	var err error
	netlibDNSNetwork, err = pool.CreateNetwork(netlibNetworkName, func(options *docker.CreateNetworkOptions) {
		options.Name = netlibNetworkName
		options.CheckDuplicate = true
		options.IPAM = &docker.IPAMOptions{
			Driver: "default",
			Config: []docker.IPAMConfig{{
				Subnet:  netlibNetworkPrefix + ".0/24",
				Gateway: netlibNetworkPrefix + ".1",
			}},
		}
		options.EnableIPv6 = false
	})
	if err != nil {
		return fmt.Errorf("could not create Network: %s:%s", netlibNetworkName, err)
	}
	netlibDNSNetworkCreated = true
	return nil
}

func buildAndRunContainer(pool *dockertest.Pool) (*dockertest.Resource, error) {
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	fmt.Printf("Try to build and start docker container %s\n", netlibDNSContainerName)
	// port := fmt.Sprintf("%d", netlibDNSPort)
	// sport := fmt.Sprintf("%d", netlibDNSSecPort)
	buildArgs := []docker.BuildArg{
		{Name: "VENDOR_IMAGE_PREFIX", Value: vendorImagePrefix},
		{Name: "BIND9_VERSION", Value: netlibRepoTag},
	}

	dockerContextDir := test.TestDir + "/docker/dns"
	return pool.BuildAndRunWithBuildOptions(
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: dockerContextDir,
			Dockerfile: "Dockerfile",
		},
		&dockertest.RunOptions{
			Hostname:     netlibDNSContainerName,
			Name:         netlibDNSContainerName,
			Networks:     []*dockertest.Network{netlibDNSNetwork},
			ExposedPorts: []string{"9053/tcp"},
			// need fixed mapping here

			PortBindings: map[docker.Port][]docker.PortBinding{
				"9053/tcp": {
					{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", netlibDNSPort)},
				},
			},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = false
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})
}

func validateContainerIP(container *dockertest.Resource) string {
	ip := container.GetIPInNetwork(netlibDNSNetwork)
	fmt.Printf("NetDNS Container IP: %s\n", ip)
	return ip
}

// func waitForDNSServer(pool *dockertest.Pool, container *dockertest.Resource) error {
func waitForDNSServer(pool *dockertest.Pool, ip string) error {
	pool.MaxWait = netlibDNSContainerTimeout * time.Second
	start := time.Now()
	err := pool.Retry(func() error {
		c, err := net.Dial("udp", net.JoinHostPort(ip, fmt.Sprintf("%d", netlibDNSPort)))
		if err != nil {
			fmt.Printf("Err:%s\n", err)
			return err
		}
		_ = c.Close()
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not connect to Net DNS Container: %v", err)
	}

	time.Sleep(10 * time.Second)
	elapsed := time.Since(start)
	fmt.Println("Net DNS Container is ready after ", elapsed.Round(time.Millisecond))

	// netlibDNSServer = server
	return nil
}

func testDNSResolution() error {
	dns := NewResolver(netlibDNSServer, netlibDNSPort, true)
	dns.IPv4Only = true
	s := "/udp"
	if dns.TCP {
		s = "/tcp"
	}
	fmt.Printf("resolve on %s:%d%s\n", dns.Nameserver, dns.Port, s)
	ips, err := dns.LookupIP(netlibTestAddr)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("could not resolve DNS for %s: %v", netlibTestAddr, err)
	}
	fmt.Printf("Host %s resolved to %s\n", netlibTestAddr, ips[0])
	return nil
}

func destroyDNSContainer(container *dockertest.Resource) {
	if container != nil {
		common.DestroyDockerContainer(container)
	}

	if netlibDNSNetworkCreated && netlibDNSNetwork != nil {
		_ = netlibDNSNetwork.Close()
	}
}
