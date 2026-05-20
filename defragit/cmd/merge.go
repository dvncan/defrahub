package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var mergeCmd = &cobra.Command{
	Use:   "merge <repo> <branch>",
	Short: "Merge another branch into the current branch",
	Args:  cobra.ExactArgs(2),
	RunE:  runMerge,
}

func runMerge(_ *cobra.Command, args []string) error {
	repoName := args[0]
	mergeBranch := args[1]

	r, err := repo.Open(context.Background(), repoName)
	if err != nil {
		return err
	}
	defer r.Close(context.Background())

	if err := db.EnsureSchema(context.Background(), r.Node); err != nil {
		return err
	}

	ctx := context.Background()
	currentBranch := r.CurrentBranch()

	ourBr, err := repo.GetBranch(ctx, r.Node, r.RepoID(), currentBranch)
	if err != nil {
		return err
	}
	theirBr, err := repo.GetBranch(ctx, r.Node, r.RepoID(), mergeBranch)
	if err != nil {
		return err
	}
	if theirBr == nil {
		return fmt.Errorf("branch %q not found", mergeBranch)
	}

	ourHead := ""
	if ourBr != nil {
		ourHead = ourBr.HeadCID
	}
	theirHead := theirBr.HeadCID

	// Fast-forward check: if our head is empty or their head is our ancestor.
	if ourHead == "" {
		if err := repo.UpdateBranchHead(ctx, r.Node, r.RepoID(), currentBranch, theirHead); err != nil {
			return err
		}
		fmt.Printf("Fast-forward to \033[36m%s\033[0m\n", repo.ShortHash(theirHead))
		return nil
	}

	// Find common ancestor.
	baseCommitID, err := repo.FindCommonAncestor(ctx, r.Node, ourHead, theirHead)
	if err != nil {
		return err
	}

	ourTree, err := repo.TreeEntryMap(ctx, r.Node, ourHead)
	if err != nil {
		return err
	}
	theirTree, err := repo.TreeEntryMap(ctx, r.Node, theirHead)
	if err != nil {
		return err
	}
	baseTree, err := repo.TreeEntryMap(ctx, r.Node, baseCommitID)
	if err != nil {
		return err
	}

	result, err := repo.ThreeWayMerge(ctx, r.Node, baseTree, ourTree, theirTree)
	if err != nil {
		return fmt.Errorf("merging: %w", err)
	}

	if result.HasConflicts() {
		// Write conflict files to disk.
		for path, content := range result.Conflicts {
			if err := os.WriteFile(path, content, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing conflict file %s: %v\n", path, err)
			}
		}
		fmt.Printf("\033[31mMerge conflict in:\033[0m\n")
		for path := range result.Conflicts {
			fmt.Printf("  %s\n", path)
		}
		fmt.Println("\nResolve conflicts, then commit.")
		return nil
	}

	// Auto-merge: stage all merged files and create a merge commit.
	for path, content := range result.Merged {
		blob, err := repo.StoreBytes(ctx, r.Node, content)
		if err != nil {
			return fmt.Errorf("storing merged blob for %s: %w", path, err)
		}
		if err := repo.StageFile(ctx, r.Node, r.RepoID(), path, blob.ContentHash); err != nil {
			return fmt.Errorf("staging merged file %s: %w", path, err)
		}
	}

	staged, _ := repo.ListIndex(ctx, r.Node, r.RepoID())
	mergeMsg := fmt.Sprintf("Merge branch '%s' into '%s'", mergeBranch, currentBranch)
	authorName := r.Config.User.Name
	authorEmail := r.Config.User.Email
	if authorName == "" {
		authorName = "Unknown"
	}
	if authorEmail == "" {
		authorEmail = "unknown@example.com"
	}

	commitID, err := repo.CreateCommit(
		ctx, r.Node,
		r.RepoID(), currentBranch, ourHead,
		mergeMsg, authorName, authorEmail,
		staged,
	)
	if err != nil {
		return fmt.Errorf("creating merge commit: %w", err)
	}
	if err := repo.ClearIndex(ctx, r.Node, r.RepoID()); err != nil {
		return err
	}

	fmt.Printf("Merged \033[36m%s\033[0m into \033[36m%s\033[0m\n", mergeBranch, currentBranch)
	fmt.Printf("[\033[36m%s\033[0m] %s\n", repo.ShortHash(commitID), mergeMsg)
	return nil
}
