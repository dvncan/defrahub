package cmd

import (
	"context"
	"fmt"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var (
	logBranch string
	logLimit  int
)

var logCmd = &cobra.Command{
	Use:   "log <repo>",
	Short: "Show commit history",
	Args:  cobra.ExactArgs(1),
	RunE:  runLog,
}

func init() {
	logCmd.Flags().StringVar(&logBranch, "branch", "", "branch to log (default: current branch)")
	logCmd.Flags().IntVar(&logLimit, "limit", 0, "max commits to show (0 = all)")
}

func runLog(_ *cobra.Command, args []string) error {
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
	branch := logBranch
	if branch == "" {
		branch = r.CurrentBranch()
	}

	br, err := repo.GetBranch(ctx, r.Node, r.RepoID(), branch)
	if err != nil {
		return err
	}
	if br == nil || br.HeadCID == "" {
		fmt.Printf("No commits on branch \033[36m%s\033[0m yet.\n", branch)
		return nil
	}

	commits, err := repo.LogCommits(ctx, r.Node, br.HeadCID, logLimit)
	if err != nil {
		return err
	}

	for _, c := range commits {
		fmt.Printf("\033[33mcommit %s\033[0m\n", c.DocID)
		fmt.Printf("Author: %s <%s>\n", c.AuthorName, c.AuthorEmail)
		fmt.Printf("Date:   %s\n", c.Timestamp)
		fmt.Printf("\n    %s\n\n", c.Message)
	}
	return nil
}
