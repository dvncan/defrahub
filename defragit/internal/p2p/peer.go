package p2p

import (
	"context"
	"fmt"
	"strings"

	"github.com/sourcenetwork/defradb/node"
)

// GetPeerID retrieves the libp2p peer ID of the running node.
// The node must have been started with P2P enabled.
func GetPeerID(ctx context.Context, n *node.Node) (string, error) {
	addrs, err := n.DB.PeerInfo(ctx)
	if err != nil {
		return "", fmt.Errorf("getting peer info: %w", err)
	}
	for _, addr := range addrs {
		// Multiaddrs look like: /ip4/127.0.0.1/tcp/9171/p2p/12D3KooW...
		// Extract the peer ID after the final /p2p/ segment.
		if idx := strings.LastIndex(addr, "/p2p/"); idx >= 0 {
			return addr[idx+5:], nil
		}
	}
	return "", fmt.Errorf("no peer ID found in peer info: %v", addrs)
}

// MultiaddrsForDisplay returns the human-readable multiaddrs for this node.
func MultiaddrsForDisplay(ctx context.Context, n *node.Node) ([]string, error) {
	return n.DB.PeerInfo(ctx)
}
