package cmd

import (
	"context"
	"fmt"

	"github.com/duncanbrown/defragit/config"
	"github.com/spf13/cobra"
)

var (
	sharePeer   string
	shareAccess string
)

var shareCmd = &cobra.Command{
	Use:   "share <repo>",
	Short: "Manage repository sharing (ACP stub)",
}

var shareAddCmd = &cobra.Command{
	Use:   "add <repo>",
	Short: "Share a repo with a peer",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		cfg, err := config.Load(args[0])
		if err != nil {
			return err
		}
		if err := ACP.ShareRepo(context.Background(), cfg.Repo.ID, shareAccess, sharePeer); err != nil {
			return err
		}
		fmt.Printf("\033[33m[ACP stub]\033[0m Access control not enforced yet.\n")
		fmt.Printf("Share this address with your collaborator:\n")
		fmt.Printf("  defragit remote add origin /ip4/<your-ip>/tcp/%d/p2p/%s\n",
			cfg.DB.P2PPort, cfg.Identity.PeerID)
		return nil
	},
}

var shareListCmd = &cobra.Command{
	Use:   "list <repo>",
	Short: "List shares for a repo",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		cfg, err := config.Load(args[0])
		if err != nil {
			return err
		}
		shares, err := ACP.ListShares(context.Background(), cfg.Repo.ID)
		if err != nil {
			return err
		}
		if len(shares) == 0 {
			fmt.Printf("\033[33m[ACP stub]\033[0m No shares configured (ACP not yet enforced).\n")
			return nil
		}
		for _, s := range shares {
			fmt.Printf("%-20s %s\n", s.Actor, s.Relation)
		}
		return nil
	},
}

var shareRevokeCmd = &cobra.Command{
	Use:   "revoke <repo>",
	Short: "Revoke a peer's access",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		cfg, err := config.Load(args[0])
		if err != nil {
			return err
		}
		if err := ACP.RevokeRepo(context.Background(), cfg.Repo.ID, "reader", sharePeer); err != nil {
			return err
		}
		fmt.Printf("\033[33m[ACP stub]\033[0m Revoked (no-op until ACP is enforced).\n")
		fmt.Printf("Peer \033[36m%s\033[0m removed from share list.\n", sharePeer)
		return nil
	},
}

func init() {
	shareAddCmd.Flags().StringVar(&sharePeer, "peer", "", "peer ID to share with")
	shareAddCmd.Flags().StringVar(&shareAccess, "access", "reader", "access level: reader|writer")
	_ = shareAddCmd.MarkFlagRequired("peer")

	shareRevokeCmd.Flags().StringVar(&sharePeer, "peer", "", "peer ID to revoke")
	_ = shareRevokeCmd.MarkFlagRequired("peer")

	shareCmd.AddCommand(shareAddCmd, shareListCmd, shareRevokeCmd)
}
