package cmd

import (
	"context"
	"fmt"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var commitMessage string

var commitCmd = &cobra.Command{
	Use:   "commit <repo>",
	Short: "Record staged changes as a new commit",
	Args:  cobra.ExactArgs(1),
	RunE:  runCommit,
}

func init() {
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "commit message")
	_ = commitCmd.MarkFlagRequired("message")
}

func runCommit(_ *cobra.Command, args []string) error {
	repoName := args[0]
	if commitMessage == "" {
		return fmt.Errorf("commit message is required (-m)")
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
	if len(staged) == 0 {
		fmt.Println("Nothing to commit — stage files with `defragit add`.")
		return nil
	}

	branch := r.CurrentBranch()
	br, err := repo.GetBranch(ctx, r.Node, r.RepoID(), branch)
	if err != nil {
		return err
	}

	parentCID := ""
	if br != nil {
		parentCID = br.HeadCID
	}

	authorName := r.Config.User.Name
	authorEmail := r.Config.User.Email
	if authorName == "" {
		authorName = "Unknown"
	}
	if authorEmail == "" {
		authorEmail = "unknown@example.com"
	}

	commitDocID, err := repo.CreateCommit(
		ctx, r.Node,
		r.RepoID(), branch, parentCID,
		commitMessage, authorName, authorEmail,
		staged,
	)
	if err != nil {
		return fmt.Errorf("creating commit: %w", err)
	}

	if err := repo.ClearIndex(ctx, r.Node, r.RepoID()); err != nil {
		return fmt.Errorf("clearing index: %w", err)
	}

	ins, del := countStagedChanges(staged)
	fmt.Printf("[\033[36m%s\033[0m \033[1m%s\033[0m] %s\n",
		branch, repo.ShortHash(commitDocID), commitMessage)
	fmt.Printf("%d files changed, %d insertions(+), %d deletions(-)\n",
		len(staged), ins, del)
	return nil
}

func countStagedChanges(staged []repo.IndexEntry) (ins, del int) {
	// Simplified: count staged files as insertions.
	return len(staged), 0
}
