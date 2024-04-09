package netlib

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const netlibDNSContainerTimeout = 10
const netlibNetworkName = "netlib-dns"
const netlibNetworkPrefix = "172.25.0"
const netlibDomain = "netlib.lan"
const netlibTestAddr = tDB

var netlibDNSContainerName string
var netlibDNSContainer *dockertest.Resource
var netlibDNSNetwork *dockertest.Network
var netlibDNSNetworkCreated = false
var netlibDNSServer = ""
var netlibDNSPort = 0

// prepareNetlibDNSContainer create a Bind9 Docker Container
func prepareNetlibDNSContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_DNS") != "" {
		err = fmt.Errorf("skipping DNS Container in CI environment")
		return
	}
	netlibDNSContainerName = os.Getenv("DNS_CONTAINER_NAME")
	if netlibDNSContainerName == "" {
		netlibDNSContainerName = "netlib-bind9"
	}
	var pool *dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}
	networks, err := pool.NetworksByName(netlibNetworkName)
	if err != nil || len(networks) == 0 {
		netlibDNSNetwork, err = pool.CreateNetwork(netlibNetworkName, func(options *docker.CreateNetworkOptions) {
			options.Name = netlibNetworkName
			options.CheckDuplicate = true
			options.IPAM = &docker.IPAMOptions{
				Driver: "default",
				Config: []docker.IPAMConfig{{
					Subnet:  netlibNetworkPrefix + ".0/16",
					Gateway: netlibNetworkPrefix + ".1",
				}},
			}
			options.EnableIPv6 = false
			// options.Internal = true
		})
		if err != nil {
			err = fmt.Errorf("could not create Network: %s:%s", netlibNetworkName, err)
			return
		}
		netlibDNSNetworkCreated = true
	} else {
		netlibDNSNetwork = &networks[0]
	}

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")

	fmt.Printf("Try to build and start docker container  %s\n", netlibDNSContainerName)
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
	dockerContextDir := test.TestDir + "/docker/dns"
	container, err = pool.BuildAndRunWithBuildOptions(
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: dockerContextDir,
			Dockerfile: "Dockerfile",
		},
		&dockertest.RunOptions{
			Hostname:     netlibDNSContainerName,
			Name:         netlibDNSContainerName,
			Networks:     []*dockertest.Network{netlibDNSNetwork},
			ExposedPorts: []string{"53/tcp", "53/udp", "953/tcp"},

			/*
				// need fixed mapping here
				PortBindings: map[docker.Port][]docker.PortBinding{
					"53/tcp": {
						{HostIP: "0.0.0.0", HostPort: netlibDNSPort},
					},

					"53/udp": {
						{HostIP: "0.0.0.0", HostPort: netlibDNSPort},
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
		err = fmt.Errorf("error starting dns docker container: %v", err)
		return
	}
	// ip := container.Container.NetworkSettings.Networks[netlibNetworkName].IPAddress
	ip := container.GetIPInNetwork(netlibDNSNetwork)
	if ip != "172.25.0.2" {
		err = fmt.Errorf("internal ip not as expected: %s", ip)
		return
	}
	pool.MaxWait = netlibDNSContainerTimeout * time.Second
	netlibDNSServer, netlibDNSPort = common.GetContainerHostAndPort(container, "53/tcp")
	fmt.Printf("Wait to successfully connect to DNS to %s:%d (max %ds)...\n", netlibDNSServer, netlibDNSPort, netlibDNSContainerTimeout)
	start := time.Now()
	var c net.Conn
	if err = pool.Retry(func() error {
		c, err = net.Dial("tcp", fmt.Sprintf("%s:%d", netlibDNSServer, netlibDNSPort))
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
	// test dns
	dns := NewResolver(netlibDNSServer, netlibDNSPort, true)
	ips, e := dns.Resolver.LookupHost(context.Background(), netlibTestAddr)
	if e != nil || len(ips) == 0 {
		fmt.Printf("Could not resolve DNS with %s: %v", netlibTestAddr, e)
		return
	}
	fmt.Println("DNS Container is ready, host", netlibTestAddr, "resolved to", ips[0])
	err = nil
	return
}

func destroyDNSContainer(container *dockertest.Resource) {
	common.DestroyDockerContainer(container)
	if netlibDNSNetworkCreated {
		_ = netlibDNSNetwork.Close()
	}
}
