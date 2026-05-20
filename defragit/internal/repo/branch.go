package repo

import (
	"context"
	"fmt"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/sourcenetwork/defradb/node"
)

// BranchInfo represents a branch record.
type BranchInfo struct {
	DocID     string
	RepoID    string
	Name      string
	HeadCID   string
	IsDefault bool
}

// CreateBranch creates a new branch record.
func CreateBranch(ctx context.Context, n *node.Node, repoID, name, headCID string, isDefault bool) error {
	mutation := fmt.Sprintf(`mutation {
		add_DG__Branch(input: {
			repoID:    %q,
			name:      %q,
			headCID:   %q,
			isDefault: %v
		}) {
			_docID
		}
	}`, repoID, name, headCID, isDefault)

	result := n.DB.ExecRequest(ctx, mutation)
	if len(result.GQL.Errors) > 0 {
		return fmt.Errorf("creating branch %q: %v", name, result.GQL.Errors)
	}
	return nil
}

// GetBranch retrieves a branch by repo + name.
func GetBranch(ctx context.Context, n *node.Node, repoID, name string) (*BranchInfo, error) {
	query := fmt.Sprintf(`query {
		DG__Branch(filter: {_and: [
			{repoID: {_eq: %q}},
			{name:   {_eq: %q}}
		]}) {
			_docID
			repoID
			name
			headCID
			isDefault
		}
	}`, repoID, name)
	result := n.DB.ExecRequest(ctx, query)
	docs, err := db.ExtractDocs(result, "DG__Branch")
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	return branchFromDoc(docs[0]), nil
}

// ListBranches returns all branches for a repo.
func ListBranches(ctx context.Context, n *node.Node, repoID string) ([]BranchInfo, error) {
	query := fmt.Sprintf(`query {
		DG__Branch(filter: {repoID: {_eq: %q}}) {
			_docID
			repoID
			name
			headCID
			isDefault
		}
	}`, repoID)
	result := n.DB.ExecRequest(ctx, query)
	docs, err := db.ExtractDocs(result, "DG__Branch")
	if err != nil {
		return nil, err
	}
	branches := make([]BranchInfo, 0, len(docs))
	for _, d := range docs {
		branches = append(branches, *branchFromDoc(d))
	}
	return branches, nil
}

// UpdateBranchHead sets the headCID of a branch to the new commit docID.
func UpdateBranchHead(ctx context.Context, n *node.Node, repoID, branchName, commitDocID string) error {
	br, err := GetBranch(ctx, n, repoID, branchName)
	if err != nil {
		return err
	}
	if br == nil {
		return fmt.Errorf("branch %q not found", branchName)
	}
	mutation := fmt.Sprintf(`mutation {
		update_DG__Branch(docID: %q, input: {headCID: %q}) {
			_docID
		}
	}`, br.DocID, commitDocID)
	result := n.DB.ExecRequest(ctx, mutation)
	if len(result.GQL.Errors) > 0 {
		return fmt.Errorf("updating branch head: %v", result.GQL.Errors)
	}
	return nil
}

// DeleteBranch removes a branch record.
func DeleteBranch(ctx context.Context, n *node.Node, repoID, name string) error {
	br, err := GetBranch(ctx, n, repoID, name)
	if err != nil {
		return err
	}
	if br == nil {
		return fmt.Errorf("branch %q not found", name)
	}
	mutation := fmt.Sprintf(`mutation {
		delete_DG__Branch(docID: %q) {
			_docID
		}
	}`, br.DocID)
	result := n.DB.ExecRequest(ctx, mutation)
	if len(result.GQL.Errors) > 0 {
		return fmt.Errorf("deleting branch: %v", result.GQL.Errors)
	}
	return nil
}

func branchFromDoc(d map[string]any) *BranchInfo {
	return &BranchInfo{
		DocID:     db.Str(d, "_docID"),
		RepoID:    db.Str(d, "repoID"),
		Name:      db.Str(d, "name"),
		HeadCID:   db.Str(d, "headCID"),
		IsDefault: db.Bool(d, "isDefault"),
	}
}