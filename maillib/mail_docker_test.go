package maillib

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

const mailRepo = "docker.io/mailserver/docker-mailserver"
const mailRepoTag = "11.3.1"
const smtpPort = 31025
const imapPort = 31143
const sslPort = 31465
const tlsPort = 31587
const imapsPort = 31993
const containerTimeout = 120

var mailContainerName string
var mailContainer *dockertest.Resource
var mailServer = "127.0.0.1"

// prepareContainer create an Oracle Docker Container
func prepareMailContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_MAIL") != "" {
		err = fmt.Errorf("skipping Mail Container in CI environment")
		return
	}
	// align hostname in CI
	host := os.Getenv("MAIL_HOST")
	if host != "" {
		mailServer = host
	}
	mailContainerName = os.Getenv("MAIL_CONTAINER_NAME")
	if mailContainerName == "" {
		mailContainerName = "mailserver"
	}
	pool, err := common.GetDockerPool()
	if err != nil {
		err = fmt.Errorf("cannot attach to docker: %v", err)
		return
	}

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + mailRepo
	fmt.Printf("Try to start docker container for %s:%s\n", repoString, mailRepoTag)
	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        mailRepoTag,

		Env: []string{
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
		},
		Hostname: mailHostname,
		Name:     mailContainerName,
		Mounts: []string{
			test.TestDir + "/mail/config:/tmp/docker-mailserver/",
			test.TestDir + "/mail/ssl:/tmp/custom-certs/:ro",
		},
		/*
			CapAdd: []string{
				"NET_ADMIN",
			},
		*/
		ExposedPorts: []string{"25", "143", "465", "587", "993"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"25": {
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", smtpPort)},
			},
			"143": {
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", imapPort)},
			},
			"465": {
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", sslPort)},
			},
			"587": {
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", tlsPort)},
			},
			"993": {
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", imapsPort)},
			},
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = fmt.Errorf("error starting mailserver docker container: %v", err)
		return
	}

	pool.MaxWait = containerTimeout * time.Second
	fmt.Printf("Wait to successfully connect to Mailserver with %s:%d (max %ds)...\n", mailServer, tlsPort, containerTimeout)
	start := time.Now()
	var c net.Conn
	if err = pool.Retry(func() error {
		c, err = net.Dial("tcp", fmt.Sprintf("%s:%d", mailServer, tlsPort))
		if err != nil {
			fmt.Printf("Err:%s\n", err)
		}
		return err
	}); err != nil {
		fmt.Printf("Could not connect to Mail Container: %s", err)
		return
	}
	_ = c.Close()

	// show env
	// cmdout, _, err=execCmd(container, []string{"bash", "-c", "env|sort"})

	// wait 20s to init container
	time.Sleep(20 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("Mail Container is available after %s\n", elapsed.Round(time.Millisecond))

	// test main.cf
	cmdout := ""
	cmd := []string{"/bin/ls", "-l", "/etc/postfix/*"}
	cmdout, _, err = common.ExecDockerCmd(container, cmd)
	if err != nil {
		fmt.Printf("Exec Error %s", err)
	} else {
		fmt.Printf("Cmd:%v\n %s", cmd, cmdout)
	}
	err = nil
	return
}
