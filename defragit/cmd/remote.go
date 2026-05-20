package cmd

import (
	"context"
	"fmt"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/repo/remote"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote <repo>",
	Short: "Manage remote peers",
}

var remoteAddCmd = &cobra.Command{
	Use:   "add <repo> <name> <peerAddr>",
	Short: "Add a remote",
	Args:  cobra.ExactArgs(3),
	RunE: func(_ *cobra.Command, args []string) error {
		repoName, name, peerAddr := args[0], args[1], args[2]
		r, err := repo.Open(context.Background(), repoName)
		if err != nil {
			return err
		}
		defer r.Close(context.Background())
		if err := db.EnsureSchema(context.Background(), r.Node); err != nil {
			return err
		}
		if err := remote.Add(context.Background(), r.Node, r.RepoID(), name, peerAddr); err != nil {
			return err
		}
		fmt.Printf("Added remote \033[36m%s\033[0m → %s\n", name, peerAddr)
		return nil
	},
}

var remoteListCmd = &cobra.Command{
	Use:   "list <repo>",
	Short: "List remotes",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		r, err := repo.Open(context.Background(), args[0])
		if err != nil {
			return err
		}
		defer r.Close(context.Background())
		if err := db.EnsureSchema(context.Background(), r.Node); err != nil {
			return err
		}
		remotes, err := remote.List(context.Background(), r.Node, r.RepoID())
		if err != nil {
			return err
		}
		for _, rem := range remotes {
			fmt.Printf("\033[36m%s\033[0m\t%s\n", rem.Name, rem.PeerAddr)
		}
		return nil
	},
}

var remoteRemoveCmd = &cobra.Command{
	Use:   "remove <repo> <name>",
	Short: "Remove a remote",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		r, err := repo.Open(context.Background(), args[0])
		if err != nil {
			return err
		}
		defer r.Close(context.Background())
		if err := db.EnsureSchema(context.Background(), r.Node); err != nil {
			return err
		}
		if err := remote.Remove(context.Background(), r.Node, r.RepoID(), args[1]); err != nil {
			return err
		}
		fmt.Printf("Removed remote \033[36m%s\033[0m.\n", args[1])
		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remoteAddCmd, remoteListCmd, remoteRemoveCmd)
}
