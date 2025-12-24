package common

import (
	"os"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDockerHelper(t *testing.T) {
	var pool *dockertest.Pool
	var err error
	var container *dockertest.Resource
	var server string
	var port int
	var dockerVersion string
	t.Run("Test GetDockerPool", func(t *testing.T) {
		pool, err = GetDockerPool()
		assert.NoErrorf(t, err, "GetDockerPool() should not return error")
		require.NotNil(t, pool, "GetDockerPool() should not return nil")
	})
	if pool == nil {
		t.Fatal("docker pool not available")
	}
	t.Run("Test GetDockerAPIversion", func(t *testing.T) {
		dockerVersion = GetDockerAPIVersion(pool.Client)
		assert.NotEmpty(t, dockerVersion, "GetDockerMinimalVersion() should return a version")
		t.Logf("Docker API version: %s", dockerVersion)
	})
	t.Run("Test GetVersionedDockerPool", func(t *testing.T) {
		endpoint := pool.Client.Endpoint()
		t.Logf("Docker Endpoint: %s", endpoint)
		pool, err = GetVersionedDockerPool(dockerVersion)
		assert.NoErrorf(t, err, "GetVersionedDockerPool() should not return error")
		require.NotNil(t, pool, "GetVersionedDockerPool() should not return nil")
		if pool != nil {
			assert.Equal(t, endpoint, pool.Client.Endpoint(), "GetVersionedDockerPool() should return the same endpoint")
			v, e := pool.Client.Version()
			if e == nil {
				t.Logf("running docker version: %v", v)
			}
		}
	})
	if pool == nil {
		t.Fatal("docker pool not available")
	}
	t.Run("Test GetDockerContainer", func(t *testing.T) {
		container, err = pool.Run("nginx", "latest", []string{})
		assert.NoErrorf(t, err, "Container should start without error")
		require.NotNil(t, container, "Container should not be nil")
	})
	t.Run("Test GetContainerHostAndPort", func(t *testing.T) {
		server, port = GetContainerHostAndPort(container, "80/tcp")
		t.Logf("server: %s, port: %d", server, port)
		assert.Greaterf(t, port, 30000, "GetContainerHostAndPort() should return a port >30000")
		assert.True(t, server == "localhost" || server == "docker", "GetContainerHostAndPort() should return localhost or docker as server")
	})
	t.Run("Test GetContainerHostAndPort other docker", func(t *testing.T) {
		_ = os.Setenv("DOCKER_HOST", "tcp://web:2375")
		server, port = GetContainerHostAndPort(container, "80/tcp")
		t.Logf("server: %s, port: %d", server, port)
		assert.Greaterf(t, port, 30000, "GetContainerHostAndPort() should return a port >30000")
		assert.True(t, server == "web", "GetContainerHostAndPort() should return localhost or docker as server")
	})
	t.Run("Test Exec on Container", func(t *testing.T) {
		var cmdout string
		cmd := []string{"ls", "-ld", "/etc/nginx"}
		cmdout, _, err = ExecDockerCmd(container, cmd)
		t.Logf("cmdout: %s", cmdout)
		assert.NoErrorf(t, err, "ExecDockerCmd() should not return error")
		assert.Contains(t, cmdout, "/etc/nginx", "ExecDockerCmd() should return /etc/nginx")
	})
	t.Run("Test DestroyDockerContainer", func(t *testing.T) {
		DestroyDockerContainer(container)
		_, ok := pool.ContainerByName(container.Container.Name)
		assert.False(t, ok, "Container should not be found in pool after DestroyDockerContainer()")
	})
}
