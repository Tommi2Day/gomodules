package maillib

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

const allInterfaces = "0.0.0.0"
const mailRepo = "docker.io/mailserver/docker-mailserver"
const mailRepoTag = "15.1.0"
const smtpPort = 31025
const imapPort = 31143
const sslPort = 31465
const tlsPort = 31587
const imapsPort = 31993
const containerTimeout = 120

var mailContainerName string
var mailContainer dockertest.ClosableResource
var mailServer = "127.0.0.1"

// prepareContainer create an Oracle Docker Container
func prepareMailContainer() (resource dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_MAIL") != "" {
		err = fmt.Errorf("skipping Mail Container in CI environment")
		return
	}

	mailContainerName = os.Getenv("MAIL_CONTAINER_NAME")
	if mailContainerName == "" {
		mailContainerName = "mailserver"
	}
	pool, err := common.GetDockerPool()
	if err != nil || pool == nil {
		err = fmt.Errorf("cannot attach to docker: %v", err)
		return
	}
	// align hostname in CI
	dh := common.GetDockerHost(pool.Client().DaemonHost())
	if dh != "" {
		mailServer = dh
		fmt.Printf("Docker Host: %s\n", mailServer)
	}
	host := os.Getenv("MAIL_HOST")
	if host != "" {
		mailServer = host
		fmt.Printf("MAIL_HOST variable is set: %s\n", host)
	}
	fmt.Printf("use mailserver: %s\n", mailServer)
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + mailRepo
	fmt.Printf("Try to start docker container for %s:%s\n", repoString, mailRepoTag)

	exposedPorts := []string{"25/tcp", "143/tcp", "465/tcp", "587/tcp", "993/tcp"}
	portBindings := network.PortMap{
		network.MustParsePort("25/tcp"): {
			{HostIP: netip.MustParseAddr(allInterfaces), HostPort: fmt.Sprintf("%d", smtpPort)},
		},
		network.MustParsePort("143/tcp"): {
			{HostIP: netip.MustParseAddr(allInterfaces), HostPort: fmt.Sprintf("%d", imapPort)},
		},
		network.MustParsePort("465/tcp"): {
			{HostIP: netip.MustParseAddr(allInterfaces), HostPort: fmt.Sprintf("%d", sslPort)},
		},
		network.MustParsePort("587/tcp"): {
			{HostIP: netip.MustParseAddr(allInterfaces), HostPort: fmt.Sprintf("%d", tlsPort)},
		},
		network.MustParsePort("993/tcp"): {
			{HostIP: netip.MustParseAddr(allInterfaces), HostPort: fmt.Sprintf("%d", imapsPort)},
		},
	}

	ctx := context.Background()
	resource, err = pool.Run(ctx, repoString,
		dockertest.WithTag(mailRepoTag),
		dockertest.WithEnv([]string{
			"LOG_LEVEL=debug",
			"ONE_DIR=1",
			"POSTFIX_INET_PROTOCOLS=ipv4",
			"PERMIT_DOCKER=connected-networks",
			"ENABLE_OPENDKIM=0",
			"ENABLE_OPENDMARC=0",
			"ENABLE_AMAVIS=0",
			"SSL_TYPE=manual",
			"SSL_CERT_PATH=/tmp/custom-certs/" + mailHostname + "-full.crt",
			"SSL_KEY_PATH=/tmp/custom-certs/" + mailHostname + ".key",
		}),
		dockertest.WithHostname(mailHostname),
		dockertest.WithName(mailContainerName),
		dockertest.WithMounts([]string{
			test.TestDir + "/docker/mail/config:/tmp/docker-mailserver/",
			test.TestDir + "/docker/mail/ssl:/tmp/custom-certs/:ro",
		}),
		/*
			CapAdd: []string{
				"NET_ADMIN",
			},
		*/
		dockertest.WithContainerConfig(func(config *container.Config) {
			if config.ExposedPorts == nil {
				config.ExposedPorts = network.PortSet{}
			}
			for _, p := range exposedPorts {
				config.ExposedPorts[network.MustParsePort(p)] = struct{}{}
			}
		}),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
			config.PortBindings = portBindings
		}),
	)

	if err != nil {
		err = fmt.Errorf("error starting mailserver docker container: %v", err)
		return
	}

	fmt.Printf("Wait to successfully connect to Mailserver with %s:%d (max %ds)...\n", mailServer, tlsPort, containerTimeout)
	start := time.Now()
	var c net.Conn
	if err = pool.Retry(ctx, containerTimeout*time.Second, func() error {
		c, err = net.Dial("udp", net.JoinHostPort(mailServer, fmt.Sprintf("%d", tlsPort)))
		if err != nil {
			fmt.Printf("Err:%s\n", err)
		}
		return err
	}); err != nil {
		fmt.Printf("Could not connect to Mail Container: %s", err)
		return
	}
	_ = c.Close()

	// wait 20s to init container
	time.Sleep(20 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("Mail Container is available after %s\n", elapsed.Round(time.Millisecond))

	// test main.cf
	cmdout := ""
	cmd := []string{"/bin/ls", "-l", "/etc/postfix/*"}
	cmdout, _, err = common.ExecDockerCmd(resource, cmd)
	if err != nil {
		fmt.Printf("Exec Error %s", err)
	} else {
		fmt.Printf("Cmd:%v\n %s", cmd, cmdout)
	}
	err = nil
	return
}
