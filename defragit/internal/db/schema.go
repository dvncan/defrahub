package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sourcenetwork/defradb/node"
)

const currentSchemaVersion = 1

// sdls lists all DefraGit collection SDL definitions.
var sdls = []string{
	`type DG__SchemaVersion {
		version:   Int
		appliedAt: DateTime
	}`,
	`type DG__Repo {
		name:        String
		description: String
		createdAt:   DateTime
		ownerPeerID: String
	}`,
	`type DG__Branch {
		repoID:    String
		name:      String
		headCID:   String
		isDefault: Boolean
	}`,
	`type DG__Commit {
		repoID:      String
		branchName:  String
		parentCID:   String
		message:     String
		authorName:  String
		authorEmail: String
		timestamp:   DateTime
		treeCID:     String
	}`,
	`type DG__Tree {
		commitCID: String
		path:      String
		blobCID:   String
		mode:      String
	}`,
	`type DG__Blob {
		contentHash: String
		content:     String
		size:        Int
		encoding:    String
	}`,
	`type DG__Index {
		repoID:      String
		filePath:    String
		blobCID:     String
		contentHash: String
		stagedAt:    DateTime
	}`,
	`type DG__Remote {
		repoID:   String
		name:     String
		peerID:   String
		peerAddr: String
	}`,
}

// EnsureSchema idempotently registers all DefraGit collections.
// Safe to call on every command startup.
func EnsureSchema(ctx context.Context, n *node.Node) error {
	// Check current schema version via sentinel collection.
	// If the query itself errors, collections likely don't exist yet.
	result := n.DB.ExecRequest(ctx, `query {
		DG__SchemaVersion(limit: 1) {
			_docID
			version
		}
	}`)

	if len(result.GQL.Errors) == 0 {
		docs, _ := ExtractDocs(result, "DG__SchemaVersion")
		if len(docs) > 0 {
			if int(Int64(docs[0], "version")) >= currentSchemaVersion {
				return nil // already up-to-date
			}
		}
	}

	// Register all collections, tolerating "already exists" errors.
	for _, sdl := range sdls {
		if _, err := n.DB.AddCollection(ctx, sdl); err != nil {
			if !isExistsErr(err) {
				return fmt.Errorf("registering schema: %w", err)
			}
		}
	}

	// Write version sentinel.
	now := time.Now().UTC().Format(time.RFC3339)
	versionMutation := fmt.Sprintf(`mutation {
		add_DG__SchemaVersion(input: {version: %d, appliedAt: %q}) {
			_docID
		}
	}`, currentSchemaVersion, now)
	r := n.DB.ExecRequest(ctx, versionMutation)
	// Tolerate errors here — version record may already exist.
	_ = r

	return nil
}

func isExistsErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "already exists") ||
		strings.Contains(s, "collection already") ||
		strings.Contains(s, "duplicate")
}
