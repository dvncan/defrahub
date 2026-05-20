package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/sourcenetwork/defradb/node"
)

// CommitInfo represents a commit record.
type CommitInfo struct {
	DocID       string
	RepoID      string
	BranchName  string
	ParentCID   string
	Message     string
	AuthorName  string
	AuthorEmail string
	Timestamp   string
	TreeCID     string
}

// TreeEntry represents a file snapshot in a commit.
type TreeEntry struct {
	DocID     string
	CommitCID string
	Path      string
	BlobCID   string // content hash
	Mode      string
}

// CreateCommit creates a new commit and its associated tree entries,
// then advances the branch head. Returns the new commit docID.
func CreateCommit(
	ctx context.Context,
	n *node.Node,
	repoID, branchName, parentCID, message, authorName, authorEmail string,
	staged []IndexEntry,
) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	// Phase 1: create the commit record (treeCID will be set to its own docID after creation).
	mutation := fmt.Sprintf(`mutation {
		add_DG__Commit(input: {
			repoID:      %q,
			branchName:  %q,
			parentCID:   %q,
			message:     %q,
			authorName:  %q,
			authorEmail: %q,
			timestamp:   %q,
			treeCID:     ""
		}) {
			_docID
		}
	}`, repoID, branchName, parentCID, message, authorName, authorEmail, now)

	result := n.DB.ExecRequest(ctx, mutation)
	docs, err := db.ExtractDocs(result, "add_DG__Commit")
	if err != nil {
		return "", fmt.Errorf("creating commit: %w", err)
	}
	if len(docs) == 0 {
		return "", fmt.Errorf("no docID returned from commit creation")
	}
	commitDocID := db.Str(docs[0], "_docID")

	// Phase 2: create DG__Tree entries for each staged file.
	for _, entry := range staged {
		treeMutation := fmt.Sprintf(`mutation {
			add_DG__Tree(input: {
				commitCID: %q,
				path:      %q,
				blobCID:   %q,
				mode:      "file"
			}) {
				_docID
			}
		}`, commitDocID, entry.FilePath, entry.ContentHash)

		r := n.DB.ExecRequest(ctx, treeMutation)
		if len(r.GQL.Errors) > 0 {
			return "", fmt.Errorf("creating tree entry for %s: %v", entry.FilePath, r.GQL.Errors)
		}
	}

	// Phase 3: update the commit's treeCID to point to itself (self-reference as tree anchor).
	updateMutation := fmt.Sprintf(`mutation {
		update_DG__Commit(docID: %q, input: {treeCID: %q}) {
			_docID
		}
	}`, commitDocID, commitDocID)
	r := n.DB.ExecRequest(ctx, updateMutation)
	if len(r.GQL.Errors) > 0 {
		return "", fmt.Errorf("updating commit treeCID: %v", r.GQL.Errors)
	}

	// Phase 4: update the branch head.
	if err := UpdateBranchHead(ctx, n, repoID, branchName, commitDocID); err != nil {
		return "", fmt.Errorf("advancing branch head: %w", err)
	}

	return commitDocID, nil
}

// GetCommit retrieves a commit by its docID.
func GetCommit(ctx context.Context, n *node.Node, commitDocID string) (*CommitInfo, error) {
	query := fmt.Sprintf(`query {
		DG__Commit(docIDs: [%q]) {
			_docID
			repoID
			branchName
			parentCID
			message
			authorName
			authorEmail
			timestamp
			treeCID
		}
	}`, commitDocID)
	result := n.DB.ExecRequest(ctx, query)
	docs, err := db.ExtractDocs(result, "DG__Commit")
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	return commitFromDoc(docs[0]), nil
}

// LogCommits walks the commit chain from headCID, returning up to limit commits.
// Pass limit <= 0 for no limit.
func LogCommits(ctx context.Context, n *node.Node, headCID string, limit int) ([]CommitInfo, error) {
	var commits []CommitInfo
	current := headCID
	for current != "" {
		if limit > 0 && len(commits) >= limit {
			break
		}
		c, err := GetCommit(ctx, n, current)
		if err != nil {
			return nil, err
		}
		if c == nil {
			break
		}
		commits = append(commits, *c)
		current = c.ParentCID
	}
	return commits, nil
}

// GetTreeEntries returns all DG__Tree entries for a given commit docID.
func GetTreeEntries(ctx context.Context, n *node.Node, commitDocID string) ([]TreeEntry, error) {
	query := fmt.Sprintf(`query {
		DG__Tree(filter: {commitCID: {_eq: %q}}) {
			_docID
			commitCID
			path
			blobCID
			mode
		}
	}`, commitDocID)
	result := n.DB.ExecRequest(ctx, query)
	docs, err := db.ExtractDocs(result, "DG__Tree")
	if err != nil {
		return nil, err
	}
	entries := make([]TreeEntry, 0, len(docs))
	for _, d := range docs {
		entries = append(entries, TreeEntry{
			DocID:     db.Str(d, "_docID"),
			CommitCID: db.Str(d, "commitCID"),
			Path:      db.Str(d, "path"),
			BlobCID:   db.Str(d, "blobCID"),
			Mode:      db.Str(d, "mode"),
		})
	}
	return entries, nil
}

// TreeEntryMap builds a map[path]contentHash from a commit's tree entries.
func TreeEntryMap(ctx context.Context, n *node.Node, commitDocID string) (map[string]string, error) {
	if commitDocID == "" {
		return map[string]string{}, nil
	}
	entries, err := GetTreeEntries(ctx, n, commitDocID)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Path] = e.BlobCID
	}
	return m, nil
}

func commitFromDoc(d map[string]any) *CommitInfo {
	return &CommitInfo{
		DocID:       db.Str(d, "_docID"),
		RepoID:      db.Str(d, "repoID"),
		BranchName:  db.Str(d, "branchName"),
		ParentCID:   db.Str(d, "parentCID"),
		Message:     db.Str(d, "message"),
		AuthorName:  db.Str(d, "authorName"),
		AuthorEmail: db.Str(d, "authorEmail"),
		Timestamp:   db.Str(d, "timestamp"),
		TreeCID:     db.Str(d, "treeCID"),
	}
}