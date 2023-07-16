package common

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ory/dockertest/v3"
)

// DockerPool is a docker pool resource
var dockerpool *dockertest.Pool

// GetDockerPool initializes a docker pool
func GetDockerPool() (*dockertest.Pool, error) {
	var err error
	if dockerpool == nil {
		dockerpool, err = dockertest.NewPool("")
		if err != nil {
			err = fmt.Errorf("cannot attach to docker: %v", err)
			return nil, err
		}
	}
	err = dockerpool.Client.Ping()
	if err != nil {
		err = fmt.Errorf("could not connect to Docker: %s", err)
		return nil, err
	}
	return dockerpool, nil
}

// GetContainerHostAndPort returns the mapped host and port of a docker container for a given portID
func GetContainerHostAndPort(container *dockertest.Resource, portID string) (server string, port int) {
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
func DestroyDockerContainer(container *dockertest.Resource) {
	if err := dockerpool.Purge(container); err != nil {
		fmt.Printf("Could not purge resource: %s\n", err)
	}
}

// ExecDockerCmd  executes an OS cmd within container and print output
func ExecDockerCmd(container *dockertest.Resource, cmd []string) (out string, code int, err error) {
	var cmdout bytes.Buffer
	cmdout.Reset()
	code, err = container.Exec(cmd, dockertest.ExecOptions{StdOut: &cmdout})
	out = cmdout.String()
	return
}
