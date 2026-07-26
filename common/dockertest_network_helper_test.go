package common

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
	"github.com/ory/dockertest/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDockerClient implements the internal dockertest.DockerClient interface
// (structurally, via github.com/ory/dockertest/v4.WithMobyClient) so network
// behavior can be tested without a running docker daemon.
type mockDockerClient struct {
	networkCreateFunc  func(ctx context.Context, name string, options mobyclient.NetworkCreateOptions) (mobyclient.NetworkCreateResult, error)
	networkInspectFunc func(ctx context.Context, networkID string, options mobyclient.NetworkInspectOptions) (mobyclient.NetworkInspectResult, error)
	networkRemoveFunc  func(ctx context.Context, networkID string, options mobyclient.NetworkRemoveOptions) (mobyclient.NetworkRemoveResult, error)
	networkListFunc    func(ctx context.Context, options mobyclient.NetworkListOptions) (mobyclient.NetworkListResult, error)

	removedNetworkIDs []string
}

func (m *mockDockerClient) NetworkCreate(ctx context.Context, name string, options mobyclient.NetworkCreateOptions) (mobyclient.NetworkCreateResult, error) {
	return m.networkCreateFunc(ctx, name, options)
}

func (m *mockDockerClient) NetworkInspect(ctx context.Context, networkID string, options mobyclient.NetworkInspectOptions) (mobyclient.NetworkInspectResult, error) {
	return m.networkInspectFunc(ctx, networkID, options)
}

func (m *mockDockerClient) NetworkRemove(ctx context.Context, networkID string, options mobyclient.NetworkRemoveOptions) (mobyclient.NetworkRemoveResult, error) {
	m.removedNetworkIDs = append(m.removedNetworkIDs, networkID)
	if m.networkRemoveFunc != nil {
		return m.networkRemoveFunc(ctx, networkID, options)
	}
	return mobyclient.NetworkRemoveResult{}, nil
}

func (m *mockDockerClient) NetworkList(ctx context.Context, options mobyclient.NetworkListOptions) (mobyclient.NetworkListResult, error) {
	if m.networkListFunc != nil {
		return m.networkListFunc(ctx, options)
	}
	return mobyclient.NetworkListResult{}, nil
}

func (m *mockDockerClient) NetworkConnect(context.Context, string, mobyclient.NetworkConnectOptions) (mobyclient.NetworkConnectResult, error) {
	return mobyclient.NetworkConnectResult{}, nil
}

func (m *mockDockerClient) NetworkDisconnect(context.Context, string, mobyclient.NetworkDisconnectOptions) (mobyclient.NetworkDisconnectResult, error) {
	return mobyclient.NetworkDisconnectResult{}, nil
}

func (m *mockDockerClient) ContainerCreate(context.Context, mobyclient.ContainerCreateOptions) (mobyclient.ContainerCreateResult, error) {
	return mobyclient.ContainerCreateResult{}, nil
}

func (m *mockDockerClient) ContainerStart(context.Context, string, mobyclient.ContainerStartOptions) (mobyclient.ContainerStartResult, error) {
	return mobyclient.ContainerStartResult{}, nil
}

func (m *mockDockerClient) ContainerStop(context.Context, string, mobyclient.ContainerStopOptions) (mobyclient.ContainerStopResult, error) {
	return mobyclient.ContainerStopResult{}, nil
}

func (m *mockDockerClient) ContainerInspect(context.Context, string, mobyclient.ContainerInspectOptions) (mobyclient.ContainerInspectResult, error) {
	return mobyclient.ContainerInspectResult{}, nil
}

func (m *mockDockerClient) ContainerRemove(context.Context, string, mobyclient.ContainerRemoveOptions) (mobyclient.ContainerRemoveResult, error) {
	return mobyclient.ContainerRemoveResult{}, nil
}

func (m *mockDockerClient) ContainerList(context.Context, mobyclient.ContainerListOptions) (mobyclient.ContainerListResult, error) {
	return mobyclient.ContainerListResult{}, nil
}

func (m *mockDockerClient) ContainerLogs(context.Context, string, mobyclient.ContainerLogsOptions) (mobyclient.ContainerLogsResult, error) {
	return nil, nil
}

func (m *mockDockerClient) ExecCreate(context.Context, string, mobyclient.ExecCreateOptions) (mobyclient.ExecCreateResult, error) {
	return mobyclient.ExecCreateResult{}, nil
}

func (m *mockDockerClient) ExecStart(context.Context, string, mobyclient.ExecStartOptions) (mobyclient.ExecStartResult, error) {
	return mobyclient.ExecStartResult{}, nil
}

func (m *mockDockerClient) ExecInspect(context.Context, string, mobyclient.ExecInspectOptions) (mobyclient.ExecInspectResult, error) {
	return mobyclient.ExecInspectResult{}, nil
}

func (m *mockDockerClient) ExecAttach(context.Context, string, mobyclient.ExecAttachOptions) (mobyclient.ExecAttachResult, error) {
	return mobyclient.ExecAttachResult{}, nil
}

func (m *mockDockerClient) ImagePull(context.Context, string, mobyclient.ImagePullOptions) (mobyclient.ImagePullResponse, error) {
	return nil, nil
}

func (m *mockDockerClient) ImageInspect(context.Context, string, ...mobyclient.ImageInspectOption) (mobyclient.ImageInspectResult, error) {
	return mobyclient.ImageInspectResult{}, nil
}

func (m *mockDockerClient) ImageBuild(context.Context, io.Reader, mobyclient.ImageBuildOptions) (mobyclient.ImageBuildResult, error) {
	return mobyclient.ImageBuildResult{}, nil
}

func (m *mockDockerClient) ImageRemove(context.Context, string, mobyclient.ImageRemoveOptions) (mobyclient.ImageRemoveResult, error) {
	return mobyclient.ImageRemoveResult{}, nil
}

func (m *mockDockerClient) Ping(context.Context, mobyclient.PingOptions) (mobyclient.PingResult, error) {
	return mobyclient.PingResult{}, nil
}

func (m *mockDockerClient) DaemonHost() string { return "mock://docker" }
func (m *mockDockerClient) Close() error       { return nil }

// newMockPool creates a dockertest.Pool backed by a mockDockerClient, so network
// creation logic can be exercised without a running docker daemon.
func newMockPool(t *testing.T, mock *mockDockerClient) dockertest.Pool {
	t.Helper()
	pool, err := dockertest.NewPool(context.Background(), "", dockertest.WithMobyClient(mock))
	require.NoError(t, err)
	return pool
}

// fakeTB is a minimal dockertest.TestingTB implementation used to observe whether
// CloseT reports a failure, without failing the enclosing test.
type fakeTB struct {
	fatalCalled bool
	fatalMsg    string
}

func (f *fakeTB) Helper()                     {}
func (f *fakeTB) Context() context.Context    { return context.Background() }
func (f *fakeTB) Cleanup(func())              {}
func (f *fakeTB) Logf(string, ...interface{}) {}
func (f *fakeTB) Fatalf(format string, args ...interface{}) {
	f.fatalCalled = true
	f.fatalMsg = fmt.Sprintf(format, args...)
}

func TestCreateNetworkWithSubnet(t *testing.T) {
	// Initialize Docker pool
	pool, err := GetDockerPool()
	assert.NoError(t, err, "Failed to get Docker pool")

	tests := []struct {
		name        string
		networkName string
		subnet      string
		gateway     string
		expectError bool
	}{
		{
			name:        "valid subnet and gateway",
			networkName: "test-network-valid",
			subnet:      "192.168.2.0/24",
			gateway:     "192.168.2.1",
			expectError: false,
		},
		{
			name:        "invalid subnet format",
			networkName: "test-network-invalid-subnet",
			subnet:      "invalid-subnet",
			gateway:     "192.168.3.1",
			expectError: true,
		},
		{
			name:        "invalid gateway format",
			networkName: "test-network-invalid-gateway",
			subnet:      "192.168.4.0/24",
			gateway:     "invalid-gateway",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net, err := CreateNetworkWithSubnet(pool, tt.networkName, tt.subnet, tt.gateway)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, net)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, net)
				assert.NotEmpty(t, net.ID())
				assert.Equal(t, tt.networkName, net.Inspect().Name)

				// Exercise CloseT's success path using the real *testing.T,
				// which satisfies dockertest.TestingTB.
				net.CloseT(t)
			}
		})
	}
}

// TestCreateNetworkWithSubnet_ReusesExistingNetworkName verifies that a stale
// network with the same name is removed before the new one is created.
func TestCreateNetworkWithSubnet_ReusesExistingNetworkName(t *testing.T) {
	pool, err := GetDockerPool()
	require.NoError(t, err)

	const name = "test-network-reuse"

	first, err := CreateNetworkWithSubnet(pool, name, "192.168.5.0/24", "192.168.5.1")
	require.NoError(t, err)
	require.NotNil(t, first)
	t.Cleanup(func() {
		_ = first.Close(context.Background())
	})

	firstID := first.ID()

	second, err := CreateNetworkWithSubnet(pool, name, "192.168.5.0/24", "192.168.5.1")
	require.NoError(t, err)
	require.NotNil(t, second)
	t.Cleanup(func() {
		_ = second.Close(context.Background())
	})

	assert.NotEqual(t, firstID, second.ID(), "expected the stale network to be replaced by a new one")
}

func TestCreateNetworkWithSubnet_NetworkCreateError(t *testing.T) {
	wantErr := errors.New("create boom")
	mock := &mockDockerClient{
		networkCreateFunc: func(context.Context, string, mobyclient.NetworkCreateOptions) (mobyclient.NetworkCreateResult, error) {
			return mobyclient.NetworkCreateResult{}, wantErr
		},
	}
	pool := newMockPool(t, mock)

	net, err := CreateNetworkWithSubnet(pool, "test-network", "192.168.6.0/24", "192.168.6.1")

	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, net)
}

func TestCreateNetworkWithSubnet_NetworkInspectError(t *testing.T) {
	wantErr := errors.New("inspect boom")
	mock := &mockDockerClient{
		networkCreateFunc: func(context.Context, string, mobyclient.NetworkCreateOptions) (mobyclient.NetworkCreateResult, error) {
			return mobyclient.NetworkCreateResult{ID: "created-id"}, nil
		},
		networkInspectFunc: func(context.Context, string, mobyclient.NetworkInspectOptions) (mobyclient.NetworkInspectResult, error) {
			return mobyclient.NetworkInspectResult{}, wantErr
		},
	}
	pool := newMockPool(t, mock)

	net, err := CreateNetworkWithSubnet(pool, "test-network", "192.168.7.0/24", "192.168.7.1")

	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, net)
	// The network created before the failed inspect must be cleaned up.
	assert.Contains(t, mock.removedNetworkIDs, "created-id")
}

const mockNetworkID = "net-id"

func TestCreateNetworkWithSubnet_MockSuccess(t *testing.T) {
	mock := &mockDockerClient{
		networkCreateFunc: func(context.Context, string, mobyclient.NetworkCreateOptions) (mobyclient.NetworkCreateResult, error) {
			return mobyclient.NetworkCreateResult{ID: mockNetworkID}, nil
		},
		networkInspectFunc: func(context.Context, string, mobyclient.NetworkInspectOptions) (mobyclient.NetworkInspectResult, error) {
			return mobyclient.NetworkInspectResult{
				Network: network.Inspect{Network: network.Network{ID: mockNetworkID, Name: "test-network"}},
			}, nil
		},
	}
	pool := newMockPool(t, mock)

	net, err := CreateNetworkWithSubnet(pool, "test-network", "192.168.8.0/24", "192.168.8.1")

	require.NoError(t, err)
	require.NotNil(t, net)
	assert.Equal(t, mockNetworkID, net.ID())
	assert.Equal(t, "test-network", net.Inspect().Name)

	err = net.Close(context.Background())
	require.NoError(t, err)
	assert.Contains(t, mock.removedNetworkIDs, mockNetworkID)
}

func TestFixedSubnetNetwork_CloseT_Failure(t *testing.T) {
	wantErr := errors.New("remove boom")
	mock := &mockDockerClient{
		networkCreateFunc: func(context.Context, string, mobyclient.NetworkCreateOptions) (mobyclient.NetworkCreateResult, error) {
			return mobyclient.NetworkCreateResult{ID: mockNetworkID}, nil
		},
		networkInspectFunc: func(context.Context, string, mobyclient.NetworkInspectOptions) (mobyclient.NetworkInspectResult, error) {
			return mobyclient.NetworkInspectResult{Network: network.Inspect{Network: network.Network{ID: mockNetworkID}}}, nil
		},
		networkRemoveFunc: func(context.Context, string, mobyclient.NetworkRemoveOptions) (mobyclient.NetworkRemoveResult, error) {
			return mobyclient.NetworkRemoveResult{}, wantErr
		},
	}
	pool := newMockPool(t, mock)

	net, err := CreateNetworkWithSubnet(pool, "test-network", "192.168.9.0/24", "192.168.9.1")
	require.NoError(t, err)
	require.NotNil(t, net)

	tb := &fakeTB{}
	net.CloseT(tb)

	assert.True(t, tb.fatalCalled)
	assert.Contains(t, tb.fatalMsg, wantErr.Error())
}

func TestRemoveNetworksByName(t *testing.T) {
	t.Run("removes matching networks and skips others", func(t *testing.T) {
		mock := &mockDockerClient{
			networkListFunc: func(context.Context, mobyclient.NetworkListOptions) (mobyclient.NetworkListResult, error) {
				return mobyclient.NetworkListResult{
					Items: []network.Summary{
						{Network: network.Network{ID: "match-1", Name: "target"}},
						{Network: network.Network{ID: "other", Name: "not-target"}},
					},
				}, nil
			},
		}

		removeNetworksByName(context.Background(), mock, "target")

		assert.Equal(t, []string{"match-1"}, mock.removedNetworkIDs)
	})

	t.Run("returns early when NetworkList errors", func(t *testing.T) {
		mock := &mockDockerClient{
			networkListFunc: func(context.Context, mobyclient.NetworkListOptions) (mobyclient.NetworkListResult, error) {
				return mobyclient.NetworkListResult{}, errors.New("list boom")
			},
		}

		removeNetworksByName(context.Background(), mock, "target")

		assert.Empty(t, mock.removedNetworkIDs)
	})

	t.Run("no matches means no removal", func(t *testing.T) {
		mock := &mockDockerClient{
			networkListFunc: func(context.Context, mobyclient.NetworkListOptions) (mobyclient.NetworkListResult, error) {
				return mobyclient.NetworkListResult{
					Items: []network.Summary{{Network: network.Network{ID: "other", Name: "not-target"}}},
				}, nil
			},
		}

		removeNetworksByName(context.Background(), mock, "target")

		assert.Empty(t, mock.removedNetworkIDs)
	})
}
