package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/duncanbrown/defragit/config"
	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/identity"
	"github.com/duncanbrown/defragit/internal/p2p"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new DefraGit repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func runInit(_ *cobra.Command, args []string) error {
	name := "default"
	if len(args) > 0 {
		name = args[0]
	}

	if _, err := os.Stat(config.RepoDir(name)); err == nil {
		return fmt.Errorf("repo %q already exists at %s", name, config.RepoDir(name))
	}

	_, peerID, err := identity.LoadOrCreate(config.IdentityKeyPath())
	if err != nil {
		return fmt.Errorf("loading identity: %w", err)
	}

	dbPath := config.DefaultDBPath(name)
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return fmt.Errorf("creating db dir: %w", err)
	}

	ctx := context.Background()

	// Attempt P2P node first to discover DefraDB's libp2p peer ID.
	n, p2pErr := db.OpenWithP2P(ctx, dbPath, "/ip4/127.0.0.1/tcp/0")
	if p2pErr != nil {
		n, err = db.Open(ctx, dbPath)
		if err != nil {
			return fmt.Errorf("starting database: %w", err)
		}
	} else {
		if dbPeerID, pidErr := p2p.GetPeerID(ctx, n); pidErr == nil && dbPeerID != "" {
			peerID = dbPeerID
		}
	}
	defer n.Close(ctx)

	if err := db.EnsureSchema(ctx, n); err != nil {
		return fmt.Errorf("registering schema: %w", err)
	}

	repoDocID, err := repo.Create(ctx, n, name, "", peerID)
	if err != nil {
		return fmt.Errorf("creating repo record: %w", err)
	}

	if err := repo.CreateBranch(ctx, n, repoDocID, "main", "", true); err != nil {
		return fmt.Errorf("creating default branch: %w", err)
	}

	globalCfg := config.LoadGlobal()

	cfg := &config.Config{
		Repo: config.RepoConfig{Name: name, ID: repoDocID},
		User: config.UserConfig{Name: globalCfg.User.Name, Email: globalCfg.User.Email},
		DB:   config.DBConfig{Path: dbPath, Port: 9181, P2PPort: 9171},
		Branch: config.BranchConfig{
			Default: "main",
			Current: "main",
		},
		Identity: config.IdentityConfig{
			KeyPath: config.IdentityKeyPath(),
			PeerID:  peerID,
		},
	}
	if err := config.Save(name, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("\033[1mInitialized DefraGit repo %q\033[0m\n", name)
	fmt.Printf("Repo ID:  \033[36m%s\033[0m\n", repoDocID)
	fmt.Printf("Peer ID:  \033[36m%s\033[0m\n", peerID)
	fmt.Printf("Branch:   \033[36mmain\033[0m\n")
	fmt.Printf("Data dir: %s\n", config.RepoDir(name))
	return nil
}
