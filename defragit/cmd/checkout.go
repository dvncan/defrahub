package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/duncanbrown/defragit/config"
	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var checkoutCreate bool

var checkoutCmd = &cobra.Command{
	Use:   "checkout <repo> <branch>",
	Short: "Switch to a branch",
	Args:  cobra.ExactArgs(2),
	RunE:  runCheckout,
}

func init() {
	checkoutCmd.Flags().BoolVarP(&checkoutCreate, "branch", "b", false, "create branch before switching")
}

func runCheckout(_ *cobra.Command, args []string) error {
	repoName := args[0]
	branchName := args[1]

	r, err := repo.Open(context.Background(), repoName)
	if err != nil {
		return err
	}
	defer r.Close(context.Background())

	if err := db.EnsureSchema(context.Background(), r.Node); err != nil {
		return err
	}

	ctx := context.Background()

	if checkoutCreate {
		currentBranch := r.CurrentBranch()
		br, err := repo.GetBranch(ctx, r.Node, r.RepoID(), currentBranch)
		if err != nil {
			return err
		}
		headCID := ""
		if br != nil {
			headCID = br.HeadCID
		}
		if err := repo.CreateBranch(ctx, r.Node, r.RepoID(), branchName, headCID, false); err != nil {
			return fmt.Errorf("creating branch: %w", err)
		}
	} else {
		br, err := repo.GetBranch(ctx, r.Node, r.RepoID(), branchName)
		if err != nil {
			return err
		}
		if br == nil {
			return fmt.Errorf("branch %q not found (use -b to create it)", branchName)
		}
	}

	// Update HEAD in config.
	r.Config.Branch.Current = branchName
	if err := config.Save(repoName, r.Config); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}

	// Write HEAD file.
	headFile := fmt.Sprintf("%s/HEAD", config.RepoDir(repoName))
	headContent := fmt.Sprintf("ref: refs/heads/%s\n", branchName)
	if err := os.WriteFile(headFile, []byte(headContent), 0644); err != nil {
		return fmt.Errorf("writing HEAD: %w", err)
	}

	fmt.Printf("Switched to branch '\033[36m%s\033[0m'\n", branchName)
	return nil
}
