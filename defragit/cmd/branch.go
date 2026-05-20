package cmd

import (
	"context"
	"fmt"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var deleteBranch string

var branchCmd = &cobra.Command{
	Use:   "branch <repo> [name]",
	Short: "List or create branches",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runBranch,
}

func init() {
	branchCmd.Flags().StringVarP(&deleteBranch, "delete", "d", "", "delete the named branch")
}

func runBranch(_ *cobra.Command, args []string) error {
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

	if deleteBranch != "" {
		if deleteBranch == r.CurrentBranch() {
			return fmt.Errorf("cannot delete the currently active branch %q", deleteBranch)
		}
		if err := repo.DeleteBranch(ctx, r.Node, r.RepoID(), deleteBranch); err != nil {
			return err
		}
		fmt.Printf("Deleted branch \033[36m%s\033[0m.\n", deleteBranch)
		return nil
	}

	if len(args) == 2 {
		// Create branch at current HEAD.
		newName := args[1]
		currentBranch := r.CurrentBranch()
		br, err := repo.GetBranch(ctx, r.Node, r.RepoID(), currentBranch)
		if err != nil {
			return err
		}
		headCID := ""
		if br != nil {
			headCID = br.HeadCID
		}
		if err := repo.CreateBranch(ctx, r.Node, r.RepoID(), newName, headCID, false); err != nil {
			return err
		}
		fmt.Printf("Created branch \033[36m%s\033[0m.\n", newName)
		return nil
	}

	// List branches.
	branches, err := repo.ListBranches(ctx, r.Node, r.RepoID())
	if err != nil {
		return err
	}
	current := r.CurrentBranch()
	for _, b := range branches {
		if b.Name == current {
			fmt.Printf("* \033[32m%s\033[0m\n", b.Name)
		} else {
			fmt.Printf("  %s\n", b.Name)
		}
	}
	return nil
}
