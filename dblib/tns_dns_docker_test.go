package dblib

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/tommi2day/gomodules/netlib"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/tommi2day/gomodules/common"

	"github.com/tommi2day/gomodules/test"
)

const dblibDNSContainerTimeout = 10
const dblibNetworkName = "dblib-dns"

var dblibDNSContainerName string
var dblibDNSContainer *dockertest.Resource
var dblibDNSNetwork *dockertest.Network
var dblibNetworkCreated = false
var dblibDNSServer = ""
var dblibDNSPort = 0

// prepareDNSContainer create a Bind9 Docker Container
func prepareDNSContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_DNS") != "" {
		err = fmt.Errorf("skipping DNS Container in CI environment")
		return
	}
	dblibDNSContainerName = os.Getenv("DNS_CONTAINER_NAME")
	if dblibDNSContainerName == "" {
		dblibDNSContainerName = "dblib-bind9"
	}
	var pool *dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}
	networks, err := pool.NetworksByName(dblibNetworkName)
	if err != nil || len(networks) == 0 {
		dblibDNSNetwork, err = pool.CreateNetwork(dblibNetworkName, func(options *docker.CreateNetworkOptions) {
			options.Name = dblibNetworkName
			options.CheckDuplicate = true
			options.IPAM = &docker.IPAMOptions{
				Driver: "default",
				Config: []docker.IPAMConfig{{
					Subnet:  "172.24.0.0/16",
					Gateway: "172.24.0.1",
				}},
			}
			options.EnableIPv6 = false
			// options.Internal = true
		})
		if err != nil {
			err = fmt.Errorf("could not create Network: %s:%s", dblibNetworkName, err)
			return
		}
		dblibNetworkCreated = true
	} else {
		dblibDNSNetwork = &networks[0]
	}

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")

	fmt.Printf("Try to build and start docker container  %s\n", dblibDNSContainerName)
	buildArgs := []docker.BuildArg{
		{
			Name:  "VENDOR_IMAGE_PREFIX",
			Value: vendorImagePrefix,
		},
		{
			Name:  "BIND9_VERSION",
			Value: "9.18",
		},
	}
	container, err = pool.BuildAndRunWithBuildOptions(
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: test.TestDir + "/docker/oracle-dns",
			Dockerfile: "Dockerfile",
		},
		&dockertest.RunOptions{
			Hostname:     dblibDNSContainerName,
			Name:         dblibDNSContainerName,
			Networks:     []*dockertest.Network{dblibDNSNetwork},
			ExposedPorts: []string{"53/tcp", "53/udp", "953/tcp"},

			/*
				// need fixed mapping here
				PortBindings: map[docker.Port][]docker.PortBinding{
					"53/tcp": {
						{HostIP: "0.0.0.0", HostPort: dblibDNSPort},
					},

					"53/udp": {
						{HostIP: "0.0.0.0", HostPort: dblibDNSPort},
					},
					"953/tcp": {
						{HostIP: "127.0.0.1", HostPort: "953"},
					},
				},
			*/
		}, func(config *docker.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})

	if err != nil {
		err = fmt.Errorf("error starting oracle-dns docker container: %v", err)
		return
	}
	// ip := container.Container.NetworkSettings.Networks[dblibNetworkName].IPAddress
	ip := container.GetIPInNetwork(dblibDNSNetwork)
	if ip != "172.24.0.2" {
		err = fmt.Errorf("internal ip not as expected: %s", ip)
		return
	}
	pool.MaxWait = dblibDNSContainerTimeout * time.Second
	dblibDNSServer, dblibDNSPort = common.GetContainerHostAndPort(container, "53/tcp")
	fmt.Printf("Wait to successfully connect to DNS to %s:%d (max %ds)...\n", dblibDNSServer, dblibDNSPort, dblibDNSContainerTimeout)
	start := time.Now()
	var c net.Conn
	if err = pool.Retry(func() error {
		c, err = net.Dial("tcp", fmt.Sprintf("%s:%d", dblibDNSServer, dblibDNSPort))
		if err != nil {
			fmt.Printf("Err:%s\n", err)
		}
		return err
	}); err != nil {
		fmt.Printf("Could not connect to DNS Container: %d", err)
		return
	}
	_ = c.Close()

	// wait 5s to init container
	time.Sleep(5 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("DNS Container is available after %s\n", elapsed.Round(time.Millisecond))
	// test oracle-dns
	dns := netlib.NewResolver(dblibDNSServer, dblibDNSPort, true)
	ips, e := dns.Resolver.LookupHost(context.Background(), racaddr)
	if e != nil || len(ips) == 0 {
		fmt.Printf("Could not resolve DNS with %s: %v", racaddr, e)
		return
	}
	fmt.Println("DNS Container is ready, host", racaddr, "resolved to", ips[0])
	err = nil
	return
}

func destroyDNSContainer(container *dockertest.Resource) {
	common.DestroyDockerContainer(container)
	if dblibNetworkCreated {
		_ = dblibDNSNetwork.Close()
	}
}
