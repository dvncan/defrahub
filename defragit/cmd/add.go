package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/duncanbrown/defragit/internal/repo"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <repo> <file> [file...]",
	Short: "Stage files for the next commit",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runAdd,
}

func runAdd(_ *cobra.Command, args []string) error {
	repoName := args[0]
	files := args[1:]

	r, err := repo.Open(context.Background(), repoName)
	if err != nil {
		return err
	}
	defer r.Close(context.Background())

	if err := db.EnsureSchema(context.Background(), r.Node); err != nil {
		return err
	}

	if len(files) == 1 && files[0] == "." {
		var expanded []string
		err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				expanded = append(expanded, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("walking directory: %w", err)
		}
		files = expanded
	}

	ctx := context.Background()
	for _, file := range files {
		blob, err := repo.StoreFile(ctx, r.Node, file)
		if err != nil {
			return fmt.Errorf("storing %s: %w", file, err)
		}
		if err := repo.StageFile(ctx, r.Node, r.RepoID(), file, blob.ContentHash); err != nil {
			return fmt.Errorf("staging %s: %w", file, err)
		}
		fmt.Printf("\033[32mstaged:\033[0m %s\n", file)
	}
	return nil
}
