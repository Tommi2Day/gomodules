package common

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	log "github.com/sirupsen/logrus"
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
	/*
		// workaround for api error client version to old
			if os.Getenv("DOCKER_API_VERSION") == "" {
				_ = os.Setenv("DOCKER_API_VERSION", "1.44")
			}
			c,err:=docker.NewClientFromEnv()
			if err!=nil{
				return nil,err
			}
			dockerpool.Client=c
	*/
	err = dockerpool.Client.Ping()
	if err != nil {
		err = fmt.Errorf("could not connect to Docker: %s", err)
		return nil, err
	}
	return dockerpool, nil
}

// GetDockerAPIVersion returns the running supported docker API version
func GetDockerAPIVersion(client *docker.Client) (v string) {
	v = ""
	versionInfo, err := client.Version()
	if versionInfo == nil || err != nil {
		return
	}
	v = versionInfo.Get("ApiVersion")
	return
}

// GetVersionedDockerPool returns a docker pool with a specific docker version, use running version if empty
func GetVersionedDockerPool(version string) (pool *dockertest.Pool, err error) {
	var client *docker.Client
	pool, err = dockertest.NewPool("")
	if err != nil || pool == nil {
		if err != nil {
			err = fmt.Errorf("cannot attach to docker: %v", err)
		} else {
			err = fmt.Errorf("pool is nil")
		}
		return nil, err
	}
	dockerVersion := GetDockerAPIVersion(pool.Client)
	mAPI, err := docker.NewAPIVersion(dockerVersion)
	if err != nil {
		err = fmt.Errorf("error parsing minimal supported version %s: %w", dockerVersion, err)
		return
	}
	if version == "" {
		version = dockerVersion
	}
	nAPI, err := docker.NewAPIVersion(version)
	if err != nil {
		err = fmt.Errorf("error parsing version %s: %w", version, err)
		return
	}
	if nAPI.LessThan(mAPI) {
		err = fmt.Errorf("version %s is less than minimal supported version %s", version, version)
		return
	}
	endpoint := pool.Client.Endpoint()
	client, err = docker.NewVersionedClient(endpoint, version)
	if err != nil || client == nil {
		err = fmt.Errorf("cannot create docker client for version %s: %s", version, err)
		return nil, err
	}
	client.SkipServerVersionCheck = true
	err = client.Ping()
	if err != nil {
		err = fmt.Errorf("could not ping Docker endpoint %s: %s", endpoint, err)
		return nil, err
	}
	pool.Client = client
	return
}

// GetContainerHostAndPort returns the mapped host and port of a docker container for a given portID
func GetContainerHostAndPort(container *dockertest.Resource, portID string) (server string, port int) {
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
func DestroyDockerContainer(container *dockertest.Resource) {
	if container == nil || dockerpool == nil {
		return
	}
	if err := dockerpool.Purge(container); err != nil {
		fmt.Printf("Could not purge resource: %s\n", err)
	}
}

// ExecDockerCmd executes an OS cmd within container and print output
func ExecDockerCmd(container *dockertest.Resource, cmd []string) (out string, code int, err error) {
	var cmdout bytes.Buffer
	if container == nil {
		err = fmt.Errorf("container is nil")
		return
	}
	cmdout.Reset()
	code, err = container.Exec(cmd, dockertest.ExecOptions{StdOut: &cmdout})
	out = cmdout.String()
	return
}

// GetDockerHost returns the docker host from a pool
func GetDockerHost(pool *dockertest.Pool) string {
	ep := pool.Client.Endpoint()
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
