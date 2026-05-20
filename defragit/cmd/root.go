package cmd

import (
	"fmt"
	"os"

	"github.com/duncanbrown/defragit/internal/acp"
	"github.com/spf13/cobra"
)

// ACP is the global access controller. Swap for a real implementation to enable ACP.
var ACP acp.AccessController = &acp.NoopACP{}

var rootCmd = &cobra.Command{
	Use:   "defragit",
	Short: "A peer-to-peer version control system backed by DefraDB",
	Long: `DefraGit mirrors the Git mental model — repos, commits, branches, diffs, merges —
but stores everything in DefraDB's MerkleCRDT graph for decentralized P2P collaboration.`,
}

// Execute runs the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		initCmd,
		addCmd,
		statusCmd,
		diffCmd,
		commitCmd,
		logCmd,
		branchCmd,
		checkoutCmd,
		mergeCmd,
		remoteCmd,
		pushCmd,
		pullCmd,
		shareCmd,
	)
}
