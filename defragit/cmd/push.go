package cmd

import (
	"context"
	"fmt"

	"github.com/duncanbrown/defragit/config"
	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/p2p"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/duncanbrown/defragit/internal/repo/remote"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <repo> [remote] [branch]",
	Short: "Push commits to a remote peer via P2P replication",
	Args:  cobra.RangeArgs(1, 3),
	RunE:  runPush,
}

func runPush(_ *cobra.Command, args []string) error {
	repoName := args[0]
	remoteName := "origin"
	if len(args) > 1 {
		remoteName = args[1]
	}

	ctx := context.Background()
	cfg, err := config.Load(repoName)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	branchName := cfg.Branch.Current
	if branchName == "" {
		branchName = cfg.Branch.Default
	}
	if len(args) > 2 {
		branchName = args[2]
	}

	// Open with P2P enabled — required for replication.
	n, err := db.OpenWithP2P(ctx, cfg.DB.Path, "")
	if err != nil {
		return fmt.Errorf("starting P2P node: %w", err)
	}
	defer n.Close(ctx)

	if err := db.EnsureSchema(ctx, n); err != nil {
		return err
	}

	rem, err := remote.Get(ctx, n, cfg.Repo.ID, remoteName)
	if err != nil {
		return fmt.Errorf("looking up remote %q: %w", remoteName, err)
	}
	if rem == nil {
		return fmt.Errorf("remote %q not found — add it with: defragit remote add %s %s <peerAddr>", remoteName, repoName, remoteName)
	}

	fmt.Printf("Pushing \033[36m%s\033[0m → \033[36m%s\033[0m\n", branchName, remoteName)
	fmt.Printf("Remote: %s\n", rem.PeerAddr)

	if err := p2p.AddReplicator(ctx, n, rem.PeerAddr); err != nil {
		return fmt.Errorf("registering replicator: %w", err)
	}

	commitCount := 0
	if br, _ := repo.GetBranch(ctx, n, cfg.Repo.ID, branchName); br != nil && br.HeadCID != "" {
		if commits, _ := repo.LogCommits(ctx, n, br.HeadCID, 0); commits != nil {
			commitCount = len(commits)
		}
	}

	fmt.Printf("\033[32m✓\033[0m Replicator registered — %d commits queued for sync to %s\n", commitCount, rem.PeerAddr)
	return nil
}