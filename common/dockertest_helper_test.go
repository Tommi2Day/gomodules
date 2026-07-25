package common

import (
	"context"
	"os"
	"testing"

	"github.com/ory/dockertest/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDockerHelper(t *testing.T) {
	var pool dockertest.ClosablePool
	var err error
	var container dockertest.ClosableResource
	var server string
	var port int
	t.Run("Test GetDockerPool", func(t *testing.T) {
		pool, err = GetDockerPool()
		assert.NoErrorf(t, err, "GetDockerPool() should not return error")
		require.NotNil(t, pool, "GetDockerPool() should not return nil")
	})
	if pool == nil {
		t.Fatal("docker pool not available")
	}
	t.Run("Test GetDockerContainer", func(t *testing.T) {
		container, err = pool.Run(context.Background(), "nginx", dockertest.WithTag("latest"))
		assert.NoErrorf(t, err, "Container should start without error")
		require.NotNil(t, container, "Container should not be nil")
	})
	t.Run("Test GetContainerHostAndPort", func(t *testing.T) {
		server, port = GetContainerHostAndPort(container, "80/tcp")
		t.Logf("server: %s, port: %d", server, port)
		assert.Greaterf(t, port, 30000, "GetContainerHostAndPort() should return a port >30000")
		assert.True(t, server == testLocalhost || server == "127.0.0.1" || server == "docker", "GetContainerHostAndPort() should return localhost, 127.0.0.1 or docker as server")
	})
	t.Run("Test GetContainerHostAndPort other docker", func(t *testing.T) {
		_ = os.Setenv("DOCKER_HOST", "tcp://web:2375")
		server, port = GetContainerHostAndPort(container, "80/tcp")
		t.Logf("server: %s, port: %d", server, port)
		assert.Greaterf(t, port, 30000, "GetContainerHostAndPort() should return a port >30000")
		assert.True(t, server == "web", "GetContainerHostAndPort() should return localhost or docker as server")
		_ = os.Unsetenv("DOCKER_HOST")
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
		_, _, execErr := ExecDockerCmd(container, []string{"true"})
		assert.Error(t, execErr, "Container should not be reachable after DestroyDockerContainer()")
	})
}

func TestGetDockerHost(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
		expected string
	}{
		{
			name:     "Valid TCP IP endpoint",
			endpoint: "tcp://192.168.99.100:2375",
			wantErr:  false,
			expected: "192.168.99.100",
		},
		{
			name:     "Valid TCP Host endpoint",
			endpoint: "tcp://docker:2375",
			wantErr:  false,
			expected: "docker",
		},
		{
			name:     "Invalid endpoint",
			endpoint: "tcp://localhost",
			wantErr:  true,
			expected: "",
		},
		{
			name:     "Valid Pipe endpoint",
			endpoint: "npipe://./pipe/docker_engine",
			wantErr:  false,
			expected: "127.0.0.1",
		},
		{
			name:     "Empty endpoint",
			endpoint: "",
			wantErr:  true,
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := GetDockerHost(tt.endpoint)
			if !tt.wantErr {
				assert.NotEmpty(t, val)
				assert.Equal(t, tt.expected, val)
			} else {
				assert.Empty(t, val)
			}
		})
	}
}
