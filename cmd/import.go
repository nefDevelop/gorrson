package cmd

import (
	"context"
	"log"
	"os"

	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/opml"
	"github.com/nef734/gorrson/internal/rss"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import feeds from an OPML file",
	Run: func(cmd *cobra.Command, args []string) {
		file, _ := cmd.Flags().GetString("file")
		user, _ := cmd.Flags().GetString("user")
		if file == "" {
			log.Fatal("Flag --file is required")
		}

		f, err := os.Open(file)
		if err != nil {
			log.Fatalf("Could not open file %s: %v", file, err)
		}
		defer f.Close()

		outlines, err := opml.Parse(f)
		if err != nil {
			log.Fatalf("Error parsing OPML: %v", err)
		}

		store, err := db.NewStore(dataDir, user)
		if err != nil {
			log.Fatal(err)
		}
		defer store.Close()

		parser := rss.NewParser()
		cmd.Printf("Importing %d feeds for user %s...\n", len(outlines), user)
		for _, o := range outlines {
			cmd.Printf("- Syncing: %s\n", o.XMLURL)
			if err := parser.SyncFeed(context.Background(), o.XMLURL, store); err != nil {
				cmd.Printf(" [!] Error syncing %s: %v\n", o.XMLURL, err)
			}
		}

		cmd.Println("Import complete.")
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringP("file", "f", "", "Path to the OPML file")
	importCmd.Flags().StringP("user", "n", "default", "Username for the database (used as filename)")
}
