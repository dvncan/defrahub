package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var (
	diffStaged bool
)

var diffCmd = &cobra.Command{
	Use:   "diff <repo> [file]",
	Short: "Show changes between working directory and staged/committed content",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().BoolVar(&diffStaged, "staged", false, "diff staged content against last commit")
}

func runDiff(_ *cobra.Command, args []string) error {
	repoName := args[0]
	var targetFile string
	if len(args) > 1 {
		targetFile = args[1]
	}

	r, err := repo.Open(context.Background(), repoName)
	if err != nil {
		return err
	}
	defer r.Close(context.Background())

	if err := db.EnsureSchema(context.Background(), r.Node); err != nil {
		return err
	}

	ctx := context.Background()
	staged, err := repo.ListIndex(ctx, r.Node, r.RepoID())
	if err != nil {
		return err
	}

	branch := r.CurrentBranch()
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

	hasDiff := false
	for _, entry := range staged {
		if targetFile != "" && entry.FilePath != targetFile {
			continue
		}

		if diffStaged {
			// Staged vs last committed.
			committedHash := committedTree[entry.FilePath]
			if committedHash == entry.ContentHash {
				continue
			}
			var oldContent []byte
			if committedHash != "" {
				oldContent, err = repo.ReadBlob(ctx, r.Node, committedHash)
				if err != nil {
					return err
				}
			}
			newContent, err := repo.ReadBlob(ctx, r.Node, entry.ContentHash)
			if err != nil {
				return err
			}
			result := repo.ComputeDiff(entry.FilePath, "a/"+entry.FilePath, "b/"+entry.FilePath, oldContent, newContent)
			if result.Hunks != "" {
				fmt.Print(result.Hunks)
				hasDiff = true
			}
		} else {
			// Working directory vs staged.
			diskContent, err := os.ReadFile(entry.FilePath)
			if err != nil {
				continue
			}
			diskHash := repo.ContentHash(diskContent)
			if diskHash == entry.ContentHash {
				continue
			}
			stagedContent, err := repo.ReadBlob(ctx, r.Node, entry.ContentHash)
			if err != nil {
				return err
			}
			result := repo.ComputeDiff(entry.FilePath, "a/"+entry.FilePath, "b/"+entry.FilePath, stagedContent, diskContent)
			if result.Hunks != "" {
				fmt.Print(result.Hunks)
				hasDiff = true
			}
		}
	}

	if !hasDiff {
		fmt.Println("No changes.")
	}
	return nil
}
