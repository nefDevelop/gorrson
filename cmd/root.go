package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	dataDir string
)

var rootCmd = &cobra.Command{
	Use:   "rss-cli",
	Short: "gorrson - A powerful RSS CLI utility",
	Long: `gorrson is a modern RSS feed manager for Go experts. 
It utilizes a multi-user SQLite architecture and provides both a CLI and a REST API.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	defaultDataDir := os.Getenv("GORRSON_DATA_DIR")
	if defaultDataDir == "" {
		defaultDataDir = "./data"
	}
	rootCmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", defaultDataDir, "Directory to store databases")
}
