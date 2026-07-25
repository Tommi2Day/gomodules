package netlib

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

const (
	netlibDNSContainerTimeout = 10
	netlibNetworkName         = "netlib-dns"
	netlibNetworkPrefix       = "172.25.0"
	netlibDomain              = "netlib.lan"
	netlibTestAddr            = tDB
	netlibRepoTag             = "9.21"
	netlibDNSPort             = 9053
)

var (
	netlibDNSContainerName  string
	netlibDNSContainer      dockertest.ClosableResource
	netlibDNSNetwork        dockertest.ClosableNetwork
	netlibDNSNetworkCreated = false
	netlibDNSServer         = "127.0.0.1"
)

// prepareNetlibDNSContainer create a Bind9 Docker Container
func prepareNetlibDNSContainer() (resource dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_NET_DNS") != "" {
		return nil, fmt.Errorf("skipping Net DNS Container in CI environment")
	}
	netlibDNSContainerName = getContainerName()
	pool, err := common.GetDockerPool()
	if err != nil {
		return nil, err
	}
	// setup network
	err = setupNetwork(pool)
	if err != nil {
		return nil, err
	}

	resource, err = buildAndRunContainer(pool)
	if err != nil {
		return
	}

	time.Sleep(10 * time.Second)

	ip := validateContainerIP(resource)
	if ip == "" {
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
	name := os.Getenv("NETDNS_CONTAINER_NAME")
	if name == "" {
		name = "netlib-bind9"
	}
	return name
}

func setupNetwork(pool dockertest.Pool) error {
	var err error
	netlibDNSNetwork, err = common.CreateNetworkWithSubnet(pool, netlibNetworkName, netlibNetworkPrefix+".0/24", netlibNetworkPrefix+".1")
	if err != nil {
		return fmt.Errorf("could not create Network: %s:%s", netlibNetworkName, err)
	}
	netlibDNSNetworkCreated = true
	return nil
}

func buildAndRunContainer(pool dockertest.Pool) (dockertest.ClosableResource, error) {
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	fmt.Printf("Try to build and start docker container %s\n", netlibDNSContainerName)
	repoTagArg := netlibRepoTag
	buildArgs := map[string]*string{
		"VENDOR_IMAGE_PREFIX": &vendorImagePrefix,
		"BIND9_VERSION":       &repoTagArg,
	}

	ctx := context.Background()
	dockerContextDir := test.TestDir + "/docker/dns"
	resource, err := pool.BuildAndRun(ctx, "netlib-bind9:test",
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: dockerContextDir,
			Dockerfile: "Dockerfile",
		},
		dockertest.WithHostname(netlibDNSContainerName),
		dockertest.WithName(netlibDNSContainerName),
		dockertest.WithContainerConfig(func(config *container.Config) {
			if config.ExposedPorts == nil {
				config.ExposedPorts = network.PortSet{}
			}
			config.ExposedPorts[network.MustParsePort("9053/tcp")] = struct{}{}
		}),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			config.AutoRemove = false
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
			// need fixed mapping here
			config.PortBindings = network.PortMap{
				network.MustParsePort("9053/tcp"): {
					{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: fmt.Sprintf("%d", netlibDNSPort)},
				},
			}
		}),
	)
	if err != nil {
		return nil, err
	}

	if err = resource.ConnectToNetwork(ctx, netlibDNSNetwork); err != nil {
		return nil, fmt.Errorf("could not connect container to network %s: %w", netlibNetworkName, err)
	}
	return resource, nil
}

func validateContainerIP(resource dockertest.Resource) string {
	ip := resource.GetIPInNetwork(netlibDNSNetwork)
	fmt.Printf("NetDNS Container IP: %s\n", ip)
	return ip
}

func waitForDNSServer(pool dockertest.Pool) error {
	dh := common.GetDockerHost(pool.Client().DaemonHost())
	if dh != "" {
		fmt.Printf("Docker Host: %s\n", dh)
	}
	ns := os.Getenv("DNS_HOST")
	if ns != "" {
		fmt.Printf("DNS_HOST variable was set to %s\n", ns)
	} else if dh != "" {
		ns = dh
	}
	if ns == "" {
		ns = netlibDNSServer
	}

	// use default resolver and port
	r := NewResolver("", 0, true)
	r.IPv4Only = true
	lips, err := r.LookupIP(ns)
	if err != nil || len(lips) == 0 {
		return fmt.Errorf("could not resolve DNS server IP for %s: %v", ns, err)
	}
	ip := lips[0]
	netlibDNSServer = ns
	fmt.Printf("DNS Host %s  IP resolved as %s\n", netlibDNSServer, ip)
	start := time.Now()
	err = pool.Retry(context.Background(), netlibDNSContainerTimeout*time.Second, func() error {
		c, e := net.Dial("tcp", net.JoinHostPort(netlibDNSServer, fmt.Sprintf("%d", netlibDNSPort)))
		if e != nil {
			fmt.Printf("Err:%s\n", e)
			return e
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
	return nil
}

func testDNSResolution() error {
	time.Sleep(10 * time.Second)
	dns := NewResolver(netlibDNSServer, netlibDNSPort, true)
	dns.IPv4Only = true
	s := "/udp"
	if dns.TCP {
		s = "/tcp"
	}
	fmt.Printf("resolver set to %s:%d%s\n", dns.Nameserver, dns.Port, s)
	ips, err := dns.LookupIP(netlibTestAddr)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("could not resolve DNS for %s: %v", netlibTestAddr, err)
	}
	fmt.Printf("Test host %s resolved to %s\n", netlibTestAddr, ips[0])
	return nil
}

func destroyDNSContainer(resource dockertest.ClosableResource) {
	ctx := context.Background()
	if resource != nil {
		_ = resource.Close(ctx)
	}

	if netlibDNSNetworkCreated && netlibDNSNetwork != nil {
		_ = netlibDNSNetwork.Close(ctx)
	}
}
