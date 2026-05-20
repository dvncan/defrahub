package p2p

import (
	"context"
	"fmt"

	"github.com/sourcenetwork/defradb/client/options"
	"github.com/sourcenetwork/defradb/node"
)

// dgCollections is the set of collections replicated between DefraGit peers.
var dgCollections = []string{
	"DG__Repo",
	"DG__Branch",
	"DG__Commit",
	"DG__Tree",
	"DG__Blob",
	"DG__Remote",
}

// AddReplicator registers a remote peer as a push replicator for all DefraGit collections.
// peerAddr is a full libp2p multiaddr, e.g. /ip4/192.168.1.5/tcp/9171/p2p/12D3KooW...
func AddReplicator(ctx context.Context, n *node.Node, peerAddr string) error {
	err := n.DB.AddReplicator(ctx,
		[]string{peerAddr},
		options.AddReplicator().SetCollectionNames(dgCollections),
	)
	if err != nil {
		return fmt.Errorf("adding replicator for %s: %w", peerAddr, err)
	}
	return nil
}

// ListReplicators returns all registered replicators for this node.
func ListReplicators(ctx context.Context, n *node.Node) ([]string, error) {
	reps, err := n.DB.ListReplicators(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing replicators: %w", err)
	}
	addrs := make([]string, 0, len(reps))
	for _, r := range reps {
		addrs = append(addrs, r.Addresses...)
	}
	return addrs, nil
}
