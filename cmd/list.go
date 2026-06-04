package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/nef734/gorrson/internal/db"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all news entries recorded in a user's database",
	Run: func(cmd *cobra.Command, args []string) {
		user, _ := cmd.Flags().GetString("user")
		unread, _ := cmd.Flags().GetBool("unread")

		store, err := db.NewStore(dataDir, user)
		if err != nil {
			log.Fatal(fmt.Errorf("could not open store for user %s: %w", user, err))
		}
		defer store.Close()

		// In CLI, we default to 50 latest entries
		entries, err := store.ListEntries(context.Background(), unread, "newest", 50, 0, "", "")
		if err != nil {
			log.Fatal(fmt.Errorf("failed to list entries: %w", err))
		}

		cmd.Printf("Displaying records for user: %s (Found: %d)\n", user, len(entries))
		cmd.Println("-------------------------------------------")
		for _, e := range entries {
			status := "[ ]"
			if e.IsRead {
				status = "[X]"
			}
			cmd.Printf("%s %s\n    URL: %s\n    Published: %s\n", status, e.Title, e.URL, e.PublishedAt.Format("2006-01-02 15:04"))
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("user", "n", "default", "User database identifier")
	listCmd.Flags().BoolP("unread", "u", false, "Show only unread articles")
}
