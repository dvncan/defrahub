package db

import (
	"context"
	"fmt"

	"github.com/sourcenetwork/defradb/client/options"
	"github.com/sourcenetwork/defradb/node"
)

// Open creates and starts a DefraDB node with P2P and HTTP disabled.
// Used for all local commands (add, commit, status, diff, log, branch).
func Open(ctx context.Context, storePath string) (*node.Node, error) {
	n, err := node.New(ctx,
		options.Node().
			SetDisableP2P(true).
			SetDisableAPI(true).
			Store().
				SetPath(storePath).
				Node(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating defradb node: %w", err)
	}
	if err := n.Start(ctx); err != nil {
		return nil, fmt.Errorf("starting defradb node: %w", err)
	}
	return n, nil
}

// OpenWithP2P creates and starts a DefraDB node with P2P enabled.
// Used for push, pull, share, and init (peer ID discovery).
func OpenWithP2P(ctx context.Context, storePath, listenAddr string) (*node.Node, error) {
	builder := options.Node().
		SetDisableAPI(true).
		Store().
			SetPath(storePath).
			Node()

	if listenAddr != "" {
		builder = builder.P2P().
			SetListenAddresses(listenAddr).
			Node()
	}

	n, err := node.New(ctx, builder)
	if err != nil {
		return nil, fmt.Errorf("creating defradb p2p node: %w", err)
	}
	if err := n.Start(ctx); err != nil {
		return nil, fmt.Errorf("starting defradb p2p node: %w", err)
	}
	return n, nil
}
