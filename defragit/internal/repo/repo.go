package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/duncanbrown/defragit/config"
	"github.com/duncanbrown/defragit/internal/db"
	"github.com/sourcenetwork/defradb/node"
)

// Repo represents an open DefraGit repository.
type Repo struct {
	Name   string
	Config *config.Config
	Node   *node.Node
}

// Open loads an existing repo by name and starts its DefraDB node.
func Open(ctx context.Context, name string) (*Repo, error) {
	cfg, err := config.Load(name)
	if err != nil {
		return nil, fmt.Errorf("loading config for repo %q: %w", name, err)
	}
	n, err := db.Open(ctx, cfg.DB.Path)
	if err != nil {
		return nil, fmt.Errorf("opening db for repo %q: %w", name, err)
	}
	return &Repo{Name: name, Config: cfg, Node: n}, nil
}

// Close shuts down the repo's DefraDB node.
func (r *Repo) Close(ctx context.Context) error {
	return r.Node.Close(ctx)
}

// RepoID returns the DefraDB docID of this repo.
func (r *Repo) RepoID() string {
	return r.Config.Repo.ID
}

// CurrentBranch returns the active branch name from config.
func (r *Repo) CurrentBranch() string {
	if r.Config.Branch.Current != "" {
		return r.Config.Branch.Current
	}
	return r.Config.Branch.Default
}

// Create inserts a new DG__Repo document and returns its docID.
func Create(ctx context.Context, n *node.Node, name, description, peerID string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	mutation := fmt.Sprintf(`mutation {
		add_DG__Repo(input: {
			name: %q,
			description: %q,
			createdAt: %q,
			ownerPeerID: %q
		}) {
			_docID
		}
	}`, name, description, now, peerID)

	result := n.DB.ExecRequest(ctx, mutation)
	docs, err := db.ExtractDocs(result, "add_DG__Repo")
	if err != nil {
		return "", fmt.Errorf("creating repo record: %w", err)
	}
	if len(docs) == 0 {
		return "", fmt.Errorf("no docID returned from repo creation")
	}
	return db.Str(docs[0], "_docID"), nil
}

// ShortHash returns the display-ready short hash (first 9 chars after "bae-").
func ShortHash(docID string) string {
	const prefix = "bae-"
	if len(docID) > len(prefix)+9 {
		return docID[len(prefix) : len(prefix)+9]
	}
	return docID
}
