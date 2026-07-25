package common

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
	"github.com/ory/dockertest/v4"
)

// fixedSubnetNetwork is a minimal dockertest.ClosableNetwork implementation backed
// directly by the moby client. dockertest.NetworkCreateOptions does not expose
// IPAM/subnet configuration, so a fixed subnet/gateway network has to be created and
// managed via the raw client instead.
type fixedSubnetNetwork struct {
	id      string
	inspect network.Inspect
	pool    dockertest.Pool
}

func (n *fixedSubnetNetwork) ID() string               { return n.id }
func (n *fixedSubnetNetwork) Inspect() network.Inspect { return n.inspect }

func (n *fixedSubnetNetwork) Close(ctx context.Context) error {
	_, err := n.pool.Client().NetworkRemove(ctx, n.id, mobyclient.NetworkRemoveOptions{})
	return err
}

func (n *fixedSubnetNetwork) CloseT(t dockertest.TestingTB) {
	t.Helper()
	if err := n.Close(context.WithoutCancel(t.Context())); err != nil {
		t.Fatalf("CloseT failed: %v", err)
	}
}

// CreateNetworkWithSubnet creates a docker network with a fixed subnet and gateway.
// Any stale network with the same name (e.g. left over from a previous test run) is
// removed first so the subnet can be reused deterministically.
func CreateNetworkWithSubnet(pool dockertest.Pool, name string, subnet string, gateway string) (dockertest.ClosableNetwork, error) {
	ctx := context.Background()
	client := pool.Client()

	removeNetworksByName(ctx, client, name)

	subnetPrefix, err := netip.ParsePrefix(subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet %s: %w", subnet, err)
	}
	gatewayAddr, err := netip.ParseAddr(gateway)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway %s: %w", gateway, err)
	}

	createResp, err := client.NetworkCreate(ctx, name, mobyclient.NetworkCreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{{
				Subnet:  subnetPrefix,
				Gateway: gatewayAddr,
			}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not create network %s: %w", name, err)
	}

	inspectResp, err := client.NetworkInspect(ctx, createResp.ID, mobyclient.NetworkInspectOptions{})
	if err != nil {
		_, _ = client.NetworkRemove(ctx, createResp.ID, mobyclient.NetworkRemoveOptions{})
		return nil, fmt.Errorf("could not inspect network %s: %w", name, err)
	}

	return &fixedSubnetNetwork{id: createResp.ID, inspect: inspectResp.Network, pool: pool}, nil
}

// removeNetworksByName removes any existing docker networks with the given name.
func removeNetworksByName(ctx context.Context, client interface {
	NetworkList(ctx context.Context, options mobyclient.NetworkListOptions) (mobyclient.NetworkListResult, error)
	NetworkRemove(ctx context.Context, networkID string, options mobyclient.NetworkRemoveOptions) (mobyclient.NetworkRemoveResult, error)
}, name string) {
	result, err := client.NetworkList(ctx, mobyclient.NetworkListOptions{
		Filters: mobyclient.Filters{}.Add("name", name),
	})
	if err != nil {
		return
	}
	for _, n := range result.Items {
		if n.Name != name {
			continue
		}
		_, _ = client.NetworkRemove(ctx, n.ID, mobyclient.NetworkRemoveOptions{})
	}
}
