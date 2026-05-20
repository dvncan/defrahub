package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <repo>",
	Short: "Show staged changes and untracked files",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func runStatus(_ *cobra.Command, args []string) error {
	repoName := args[0]
	r, err := repo.Open(context.Background(), repoName)
	if err != nil {
		return err
	}
	defer r.Close(context.Background())

	if err := db.EnsureSchema(context.Background(), r.Node); err != nil {
		return err
	}

	ctx := context.Background()
	branch := r.CurrentBranch()
	fmt.Printf("On branch \033[36m%s\033[0m\n\n", branch)

	// Get staged files.
	staged, err := repo.ListIndex(ctx, r.Node, r.RepoID())
	if err != nil {
		return err
	}

	// Get last committed tree for the current branch.
	br, err := repo.GetBranch(ctx, r.Node, r.RepoID(), branch)
	if err != nil {
		return err
	}

	var committedTree map[string]string
	if br != nil && br.HeadCID != "" {
		committedTree, err = repo.TreeEntryMap(ctx, r.Node, br.HeadCID)
		if err != nil {
			return err
		}
	} else {
		committedTree = map[string]string{}
	}

	stagedMap := map[string]string{}
	for _, e := range staged {
		stagedMap[e.FilePath] = e.ContentHash
	}

	if len(staged) > 0 {
		fmt.Println("Changes staged for commit:")
		for _, e := range staged {
			if _, existed := committedTree[e.FilePath]; !existed {
				fmt.Printf("  \033[32m(new file)\033[0m  %s\n", e.FilePath)
			} else if committedTree[e.FilePath] != e.ContentHash {
				fmt.Printf("  \033[32m(modified)\033[0m  %s\n", e.FilePath)
			}
		}
		fmt.Println()
	} else {
		fmt.Println("No changes staged for commit.")
		fmt.Println()
	}

	// Find untracked files (in working dir but not staged or committed).
	var untracked []string
	_ = filepath.Walk(".", func(path string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			if _, staged := stagedMap[path]; !staged {
				if _, committed := committedTree[path]; !committed {
					untracked = append(untracked, path)
				}
			}
		}
		return nil
	})

	if len(untracked) > 0 {
		fmt.Println("Untracked files:")
		for _, f := range untracked {
			fmt.Printf("  \033[33m%s\033[0m\n", f)
		}
	}

	return nil
}
