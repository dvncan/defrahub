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

var pullCmd = &cobra.Command{
	Use:   "pull <repo> [remote] [branch]",
	Short: "Pull commits from a remote peer",
	Args:  cobra.RangeArgs(1, 3),
	RunE:  runPull,
}

func runPull(_ *cobra.Command, args []string) error {
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

	// Open with P2P enabled — required to connect to the remote peer.
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

	// Record local branch head before sync.
	localHead := ""
	if localBranch, _ := repo.GetBranch(ctx, n, cfg.Repo.ID, branchName); localBranch != nil {
		localHead = localBranch.HeadCID
	}

	fmt.Printf("Pulling \033[36m%s\033[0m ← \033[36m%s\033[0m\n", branchName, remoteName)
	fmt.Printf("Remote: %s\n", rem.PeerAddr)

	// Register remote as a replication peer, which establishes the P2P connection
	// and triggers DefraDB to sync existing collection data from the remote.
	if err := p2p.AddReplicator(ctx, n, rem.PeerAddr); err != nil {
		return fmt.Errorf("connecting to remote: %w", err)
	}

	// Re-read branch head after establishing connection; DefraDB pubsub may have
	// delivered updated branch records during the connection handshake.
	br, _ := repo.GetBranch(ctx, n, cfg.Repo.ID, branchName)
	if br == nil || br.HeadCID == "" || br.HeadCID == localHead {
		fmt.Println("Already up to date.")
		return nil
	}
	remoteHead := br.HeadCID

	// Walk the remote commit chain to find whether localHead is an ancestor.
	newCount := 0
	current := remoteHead
	isAncestor := localHead == ""
	for current != "" {
		if current == localHead {
			isAncestor = true
			break
		}
		c, cErr := repo.GetCommit(ctx, n, current)
		if cErr != nil || c == nil {
			break
		}
		newCount++
		current = c.ParentCID
	}

	if !isAncestor {
		fmt.Printf("Diverged histories — run `defragit merge %s` to reconcile.\n", branchName)
		return nil
	}

	// Fast-forward: advance local branch head to remote head.
	if err := repo.UpdateBranchHead(ctx, n, cfg.Repo.ID, branchName, remoteHead); err != nil {
		return fmt.Errorf("fast-forward: %w", err)
	}

	fmt.Printf("\033[32m✓\033[0m %d new commit(s)\n", newCount)
	fmt.Printf("Fast-forward to \033[36m%s\033[0m\n", repo.ShortHash(remoteHead))
	return nil
}