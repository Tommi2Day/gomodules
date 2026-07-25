package common

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/ory/dockertest/v4"
	log "github.com/sirupsen/logrus"
)

// dockerpool is a docker pool resource
var dockerpool dockertest.ClosablePool

// GetDockerPool initializes a docker pool
func GetDockerPool() (dockertest.ClosablePool, error) {
	var err error
	if dockerpool == nil {
		dockerpool, err = dockertest.NewPool(context.Background(), "")
		if err != nil {
			err = fmt.Errorf("cannot attach to docker: %v", err)
			return nil, err
		}
	}
	return dockerpool, nil
}

// GetContainerHostAndPort returns the mapped host and port of a docker container for a given portID
func GetContainerHostAndPort(container dockertest.Resource, portID string) (server string, port int) {
	if container == nil {
		return
	}
	dockerURL := os.Getenv("DOCKER_HOST")
	containerAddress := container.GetHostPort(portID)
	s, p, _ := GetHostPort(containerAddress)
	if dockerURL == "" {
		server = s
	} else {
		// replace server with docker host
		server, _, _ = GetHostPort(dockerURL)
	}
	port = p
	return
}

// DestroyDockerContainer destroys a docker container
func DestroyDockerContainer(container dockertest.ClosableResource) {
	if container == nil {
		return
	}
	if err := container.Close(context.Background()); err != nil {
		fmt.Printf("Could not purge resource: %s\n", err)
	}
}

// ExecDockerCmd executes an OS cmd within container and print output
func ExecDockerCmd(container dockertest.Resource, cmd []string) (out string, code int, err error) {
	if container == nil {
		err = fmt.Errorf("container is nil")
		return
	}
	var res dockertest.ExecResult
	res, err = container.Exec(context.Background(), cmd)
	if err == nil && !IsNil(res) {
		out = res.StdOut
		code = res.ExitCode
	} else if IsNil(res) && IsNil(err) {
		err = fmt.Errorf("could not execute command")
		code = 128
	}
	return
}

// GetDockerHost returns the docker host from a docker daemon endpoint (e.g. pool.Client().DaemonHost())
func GetDockerHost(ep string) string {
	log.Debugf("Docker Endpoint: %s\n", ep)
	re := regexp.MustCompile("tcp://(.*):")
	re2 := regexp.MustCompile("npipe://.*")
	match := re.FindStringSubmatch(ep)
	host := ""
	switch {
	case len(match) > 1:
		host = match[1]
		log.Debugf("Docker Host: %s", host)
	case re2.MatchString(ep):
		log.Debugf("Docker Host for pipe URL: %s", host)
		host = "127.0.0.1"
	default:
		log.Debugf("Could not extract Docker Host from Endpoint: %s", ep)
	}
	return host
}
