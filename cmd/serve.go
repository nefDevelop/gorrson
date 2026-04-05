package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/nef734/gorrson/internal/api"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/rss"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the REST API and background workers for all user databases",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		if port == "" {
			port = os.Getenv("GORRSON_PORT")
			if port == "" {
				port = "7077"
			}
		}

		dataDir := os.Getenv("GORRSON_DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}

		// Ensure data directory exists
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			os.MkdirAll(dataDir, 0755)
		}

		// Start workers for all existing users found in the data directory
		files, err := os.ReadDir(dataDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".db") && file.Name() != "global.db" {
					username := strings.TrimSuffix(file.Name(), ".db")
					
					store, err := db.NewStore(dataDir, username)
					if err != nil {
						log.Printf("[Serve] Could not load database for %s: %v", username, err)
						continue
					}
					
					// Each user gets a background worker to sync their feeds independently
					worker := rss.NewWorker(store, 15*time.Minute)
					go worker.Start(context.Background())
					log.Printf("[Serve] Started background worker for user: %s", username)
				}
			}
		}

		server := api.NewServer(dataDir)
		addr := ":" + port
		fmt.Printf("gorrson server listening on %s...\n", addr)
		if err := server.Start(addr); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("port", "p", "7077", "Port to listen on")
}
