package dblib

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/netlib"
	"github.com/tommi2day/gomodules/test"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/ory/dockertest/v4"
)

const (
	dblibDNSContainerTimeout = 10
	dblibNetworkName         = "dblib-dns"
	dblibNetworkPrefix       = "172.24.0"
	dblibRepoTag             = "9.21"
	dblibDNSPort             = 9054
	dblibTestAddr            = racaddr
)

var (
	dblibDNSContainerName  string
	dblibDNSContainer      dockertest.ClosableResource
	dblibDNSNetwork        dockertest.ClosableNetwork
	dblibDNSNetworkCreated = false
	dblibDNSServer         = "127.0.0.1"
)

// prepareDBlibDNSContainer create a Bind9 Docker Container
func prepareDBlibDNSContainer() (resource dockertest.ClosableResource, err error) {
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
	name := os.Getenv("DBDNS_CONTAINER_NAME")
	if name == "" {
		name = "dblib-bind9"
	}
	return name
}

func setupNetwork(pool dockertest.Pool) error {
	var err error
	dblibDNSNetwork, err = common.CreateNetworkWithSubnet(pool, dblibNetworkName, dblibNetworkPrefix+".0/24", dblibNetworkPrefix+".1")
	if err != nil {
		return fmt.Errorf("could not create Network: %s:%s", dblibNetworkName, err)
	}
	dblibDNSNetworkCreated = true
	return nil
}

func buildAndRunContainer(pool dockertest.Pool) (dockertest.ClosableResource, error) {
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	fmt.Printf("Try to build and start docker container %s\n", dblibDNSContainerName)
	repoTagArg := dblibRepoTag
	buildArgs := map[string]*string{
		"VENDOR_IMAGE_PREFIX": &vendorImagePrefix,
		"BIND9_VERSION":       &repoTagArg,
	}

	ctx := context.Background()
	dockerContextDir := test.TestDir + "/docker/oracle-dns"
	resource, err := pool.BuildAndRun(ctx, "dblib-bind9:test",
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: dockerContextDir,
			Dockerfile: "Dockerfile",
		},
		dockertest.WithHostname(dblibDNSContainerName),
		dockertest.WithName(dblibDNSContainerName),
		dockertest.WithContainerConfig(func(config *container.Config) {
			if config.ExposedPorts == nil {
				config.ExposedPorts = network.PortSet{}
			}
			config.ExposedPorts[network.MustParsePort("9054/tcp")] = struct{}{}
		}),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			config.AutoRemove = false
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
			// need fixed mapping here
			config.PortBindings = network.PortMap{
				network.MustParsePort("9054/tcp"): {
					{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: fmt.Sprintf("%d", dblibDNSPort)},
				},
			}
		}),
	)
	if err != nil {
		return nil, err
	}

	if err = resource.ConnectToNetwork(ctx, dblibDNSNetwork); err != nil {
		return nil, fmt.Errorf("could not connect container to network %s: %w", dblibNetworkName, err)
	}
	return resource, nil
}

func validateContainerIP(resource dockertest.Resource) string {
	ip := resource.GetIPInNetwork(dblibDNSNetwork)
	fmt.Printf("DB DNS Container IP: %s\n", ip)
	return ip
}

func waitForDNSServer(pool dockertest.Pool) error {
	dh := common.GetDockerHost(pool.Client().DaemonHost())
	if dh != "" {
		fmt.Printf("Docker Host: %s\n", dh)
	}
	ns := os.Getenv("DB_HOST")
	if ns != "" {
		fmt.Printf("DB_HOST variable was set to %s\n", ns)
	} else if dh != "" {
		ns = dh
	}
	if ns == "" {
		ns = dblibDNSServer
	}

	// use default resolver and port
	r := netlib.NewResolver("", 0, true)
	r.IPv4Only = true
	lips, err := r.LookupIP(ns)
	if err != nil || len(lips) == 0 {
		return fmt.Errorf("could not resolve DNS server IP for %s: %v", ns, err)
	}
	ip := lips[0]
	dblibDNSServer = ns
	fmt.Printf("DNS Host %s  IP resolved as %s\n", dblibDNSServer, ip)
	start := time.Now()
	err = pool.Retry(context.Background(), dblibDNSContainerTimeout*time.Second, func() error {
		c, e := net.Dial("tcp", net.JoinHostPort(dblibDNSServer, fmt.Sprintf("%d", dblibDNSPort)))
		if e != nil {
			fmt.Printf("Err:%s\n", e)
			return e
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

func destroyDNSContainer(resource dockertest.ClosableResource) {
	ctx := context.Background()
	if resource != nil {
		_ = resource.Close(ctx)
	}

	if dblibDNSNetworkCreated && dblibDNSNetwork != nil {
		_ = dblibDNSNetwork.Close(ctx)
	}
}
