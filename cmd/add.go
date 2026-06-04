package cmd

import (
	"context"
	"log"

	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/rss"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a feed manually",
	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")
		user, _ := cmd.Flags().GetString("user")
		if url == "" {
			log.Fatal("Flag --url is required")
		}

		store, err := db.NewStore(dataDir, user)
		if err != nil {
			log.Fatal(err)
		}
		defer store.Close()

		parser := rss.NewParser()
		cmd.Printf("Syncing feed from: %s...\n", url)
		if err := parser.SyncFeed(context.Background(), url, store); err != nil {
			log.Fatal(err)
		}

		cmd.Printf("Feed %s added successfully for user %s\n", url, user)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("url", "u", "", "URL of the RSS feed")
	addCmd.Flags().StringP("user", "n", "default", "Username for the database (used as filename)")
}
