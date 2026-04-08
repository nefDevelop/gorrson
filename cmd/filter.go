package cmd

import (
	"log"
	"regexp"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
)

var filterCmd = &cobra.Command{
	Use:   "filter",
	Short: "Manage entry filters (BLOCK, REPLACE)",
}

var filterAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new filter",
	Run: func(cmd *cobra.Command, args []string) {
		user, _ := cmd.Flags().GetString("user")
		pattern, _ := cmd.Flags().GetString("pattern")
		action, _ := cmd.Flags().GetString("action")
		target, _ := cmd.Flags().GetString("target")
		feedID, _ := cmd.Flags().GetString("feed")
		replacement, _ := cmd.Flags().GetString("replacement")

		if pattern == "" || action == "" {
			log.Fatal("Flags --pattern and --action are required")
		}

		if _, err := regexp.Compile(pattern); err != nil {
			log.Fatalf("Invalid regex pattern: %v", err)
		}

		store, err := db.NewStore(dataDir, user)
		if err != nil {
			log.Fatal(err)
		}
		defer store.Close()

		f := &models.Filter{
			ID:          uuid.New().String(),
			FeedID:      feedID,
			Pattern:     pattern,
			Action:      action,
			Target:      target,
			Replacement: replacement,
		}

		if err := store.AddFilter(f); err != nil {
			log.Fatal(err)
		}

		cmd.Printf("Filter '%s' added successfully for user %s\n", pattern, user)
	},
}

var filterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all filters",
	Run: func(cmd *cobra.Command, args []string) {
		user, _ := cmd.Flags().GetString("user")
		store, err := db.NewStore(dataDir, user)
		if err != nil {
			log.Fatal(err)
		}
		defer store.Close()

		filters, err := store.ListFilters()
		if err != nil {
			log.Fatal(err)
		}

		cmd.Printf("Filters for user %s:\n", user)
		cmd.Printf("%-36s | %-15s | %-10s | %-10s | %s\n", "ID", "Pattern", "Action", "Target", "FeedID")
		cmd.Println(string(make([]byte, 100))) // Simple line
		for _, f := range filters {
			fid := f.FeedID
			if fid == "" {
				fid = "(GLOBAL)"
			}
			cmd.Printf("%-36s | %-15s | %-10s | %-10s | %s\n", f.ID, f.Pattern, f.Action, f.Target, fid)
		}
	},
}

var filterRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a filter by ID",
	Run: func(cmd *cobra.Command, args []string) {
		user, _ := cmd.Flags().GetString("user")
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			log.Fatal("Flag --id is required")
		}

		store, err := db.NewStore(dataDir, user)
		if err != nil {
			log.Fatal(err)
		}
		defer store.Close()

		if err := store.DeleteFilter(id); err != nil {
			log.Fatal(err)
		}
		cmd.Printf("Filter %s removed.\n", id)
	},
}

func init() {
	rootCmd.AddCommand(filterCmd)
	filterCmd.AddCommand(filterAddCmd)
	filterCmd.AddCommand(filterListCmd)
	filterCmd.AddCommand(filterRemoveCmd)

	filterAddCmd.Flags().StringP("user", "n", "default", "User")
	filterAddCmd.Flags().StringP("pattern", "p", "", "Regex or keyword to match")
	filterAddCmd.Flags().StringP("action", "a", "BLOCK", "BLOCK or REPLACE")
	filterAddCmd.Flags().StringP("target", "t", "both", "title, content, or both")
	filterAddCmd.Flags().StringP("feed", "f", "", "Feed ID (optional, default: global)")
	filterAddCmd.Flags().StringP("replacement", "r", "", "Replacement text (for REPLACE action)")

	filterListCmd.Flags().StringP("user", "n", "default", "User")
	filterRemoveCmd.Flags().StringP("user", "n", "default", "User")
	filterRemoveCmd.Flags().StringP("id", "i", "", "Filter ID to remove")
}
