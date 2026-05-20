package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/sourcenetwork/defradb/node"
)

// IndexEntry represents a staged file.
type IndexEntry struct {
	DocID       string
	RepoID      string
	FilePath    string
	BlobCID     string // content hash of the staged content
	ContentHash string
	StagedAt    string
}

// StageFile adds or replaces an index entry for filePath in the given repo.
func StageFile(ctx context.Context, n *node.Node, repoID, filePath, contentHash string) error {
	// Remove any existing index entry for this file.
	if err := unstageFile(ctx, n, repoID, filePath); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	mutation := fmt.Sprintf(`mutation {
		add_DG__Index(input: {
			repoID:      %q,
			filePath:    %q,
			blobCID:     %q,
			contentHash: %q,
			stagedAt:    %q
		}) {
			_docID
		}
	}`, repoID, filePath, contentHash, contentHash, now)

	result := n.DB.ExecRequest(ctx, mutation)
	if len(result.GQL.Errors) > 0 {
		return fmt.Errorf("staging file: %v", result.GQL.Errors)
	}
	return nil
}

// ListIndex returns all staged entries for a repo.
func ListIndex(ctx context.Context, n *node.Node, repoID string) ([]IndexEntry, error) {
	query := fmt.Sprintf(`query {
		DG__Index(filter: {repoID: {_eq: %q}}) {
			_docID
			repoID
			filePath
			blobCID
			contentHash
			stagedAt
		}
	}`, repoID)
	result := n.DB.ExecRequest(ctx, query)
	docs, err := db.ExtractDocs(result, "DG__Index")
	if err != nil {
		return nil, err
	}
	entries := make([]IndexEntry, 0, len(docs))
	for _, d := range docs {
		entries = append(entries, IndexEntry{
			DocID:       db.Str(d, "_docID"),
			RepoID:      db.Str(d, "repoID"),
			FilePath:    db.Str(d, "filePath"),
			BlobCID:     db.Str(d, "blobCID"),
			ContentHash: db.Str(d, "contentHash"),
			StagedAt:    db.Str(d, "stagedAt"),
		})
	}
	return entries, nil
}

// ClearIndex removes all staged entries for a repo (called after commit).
func ClearIndex(ctx context.Context, n *node.Node, repoID string) error {
	mutation := fmt.Sprintf(`mutation {
		delete_DG__Index(filter: {repoID: {_eq: %q}}) {
			_docID
		}
	}`, repoID)
	result := n.DB.ExecRequest(ctx, mutation)
	if len(result.GQL.Errors) > 0 {
		return fmt.Errorf("clearing index: %v", result.GQL.Errors)
	}
	return nil
}

// unstageFile removes an existing index entry for a specific file.
func unstageFile(ctx context.Context, n *node.Node, repoID, filePath string) error {
	mutation := fmt.Sprintf(`mutation {
		delete_DG__Index(filter: {_and: [
			{repoID:   {_eq: %q}},
			{filePath: {_eq: %q}}
		]}) {
			_docID
		}
	}`, repoID, filePath)
	result := n.DB.ExecRequest(ctx, mutation)
	// Tolerate "not found" errors — the entry may not exist yet.
	if len(result.GQL.Errors) > 0 {
		errStr := fmt.Sprintf("%v", result.GQL.Errors)
		if containsAny(errStr, "not found", "no documents") {
			return nil
		}
		return fmt.Errorf("unstaging file: %v", result.GQL.Errors)
	}
	return nil
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}
