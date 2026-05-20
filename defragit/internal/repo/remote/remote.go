package remote

import (
	"context"
	"fmt"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/sourcenetwork/defradb/node"
)

// RemoteInfo represents a DG__Remote record.
type RemoteInfo struct {
	DocID    string
	RepoID   string
	Name     string
	PeerID   string
	PeerAddr string
}

// Add creates a DG__Remote record.
func Add(ctx context.Context, n *node.Node, repoID, name, peerAddr string) error {
	mutation := fmt.Sprintf(`mutation {
		add_DG__Remote(input: {
			repoID:   %q,
			name:     %q,
			peerID:   "",
			peerAddr: %q
		}) {
			_docID
		}
	}`, repoID, name, peerAddr)
	result := n.DB.ExecRequest(ctx, mutation)
	if len(result.GQL.Errors) > 0 {
		return fmt.Errorf("adding remote: %v", result.GQL.Errors)
	}
	return nil
}

// List returns all remotes for a repo.
func List(ctx context.Context, n *node.Node, repoID string) ([]RemoteInfo, error) {
	query := fmt.Sprintf(`query {
		DG__Remote(filter: {repoID: {_eq: %q}}) {
			_docID
			repoID
			name
			peerID
			peerAddr
		}
	}`, repoID)
	result := n.DB.ExecRequest(ctx, query)
	docs, err := db.ExtractDocs(result, "DG__Remote")
	if err != nil {
		return nil, err
	}
	remotes := make([]RemoteInfo, 0, len(docs))
	for _, d := range docs {
		remotes = append(remotes, RemoteInfo{
			DocID:    db.Str(d, "_docID"),
			RepoID:   db.Str(d, "repoID"),
			Name:     db.Str(d, "name"),
			PeerID:   db.Str(d, "peerID"),
			PeerAddr: db.Str(d, "peerAddr"),
		})
	}
	return remotes, nil
}

// Get returns a single remote by name.
func Get(ctx context.Context, n *node.Node, repoID, name string) (*RemoteInfo, error) {
	query := fmt.Sprintf(`query {
		DG__Remote(filter: {_and: [
			{repoID: {_eq: %q}},
			{name:   {_eq: %q}}
		]}) {
			_docID
			repoID
			name
			peerID
			peerAddr
		}
	}`, repoID, name)
	result := n.DB.ExecRequest(ctx, query)
	docs, err := db.ExtractDocs(result, "DG__Remote")
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	r := &RemoteInfo{
		DocID:    db.Str(docs[0], "_docID"),
		RepoID:   db.Str(docs[0], "repoID"),
		Name:     db.Str(docs[0], "name"),
		PeerID:   db.Str(docs[0], "peerID"),
		PeerAddr: db.Str(docs[0], "peerAddr"),
	}
	return r, nil
}

// Remove deletes a remote by name.
func Remove(ctx context.Context, n *node.Node, repoID, name string) error {
	rem, err := Get(ctx, n, repoID, name)
	if err != nil {
		return err
	}
	if rem == nil {
		return fmt.Errorf("remote %q not found", name)
	}
	mutation := fmt.Sprintf(`mutation {
		delete_DG__Remote(docID: %q) {
			_docID
		}
	}`, rem.DocID)
	result := n.DB.ExecRequest(ctx, mutation)
	if len(result.GQL.Errors) > 0 {
		return fmt.Errorf("removing remote: %v", result.GQL.Errors)
	}
	return nil
}
