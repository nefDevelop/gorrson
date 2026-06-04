package cmd

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Advanced user account management",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered users",
	Run: func(cmd *cobra.Command, args []string) {
		files, err := os.ReadDir(dataDir)
		if err != nil {
			log.Fatal(err)
		}

		cmd.Println("Registered Gorrson Users:")
		cmd.Println("-------------------------")
		count := 0
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".db") && file.Name() != "global.db" {
				username := strings.TrimSuffix(file.Name(), ".db")
				cmd.Printf("- %s\n", username)
				count++
			}
		}
		cmd.Printf("\nTotal: %d users found in %s\n", count, dataDir)
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a user account and its entire database",
	Run: func(cmd *cobra.Command, args []string) {
		user, _ := cmd.Flags().GetString("user")
		if user == "" {
			log.Fatal("Flag --user is required")
		}

		dbPath := filepath.Join(dataDir, user+".db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			log.Fatalf("User '%s' does not exist.", user)
		}

		cmd.Printf("Are you sure you want to delete user '%s' and ALL their data? (y/N): ", user)
		var confirm string
		fmt.Fscanln(cmd.InOrStdin(), &confirm)
		if strings.ToLower(confirm) != "y" {
			cmd.Println("Operation cancelled.")
			return
		}

		if err := os.Remove(dbPath); err != nil {
			log.Fatal(err)
		}
		cmd.Printf("User '%s' deleted successfully.\n", user)
	},
}

var userPasswdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change a user's password",
	Run: func(cmd *cobra.Command, args []string) {
		user, _ := cmd.Flags().GetString("user")
		pass, _ := cmd.Flags().GetString("pass")
		if user == "" || pass == "" {
			log.Fatal("Flags --user and --pass are required")
		}

		hash, err := auth.HashPassword(pass)
		if err != nil {
			log.Fatal(err)
		}

		// Calculate Fever Key: bcrypt(md5(user:pass))
		feverHash := md5.Sum([]byte(user + ":" + pass))
		feverApiKey := fmt.Sprintf("%x", feverHash)
		feverBcrypt, err := auth.HashPassword(feverApiKey)
		if err != nil {
			log.Fatal(err)
		}

		store, err := db.NewStore(dataDir, user)
		if err != nil {
			log.Fatal(err)
		}
		defer store.Close()

		if err := store.UpsertUser(context.Background(), user, hash, feverBcrypt); err != nil {
			log.Fatal(err)
		}

		cmd.Printf("Password for user '%s' updated successfully.\n", user)
	},
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userPasswdCmd)

	userDeleteCmd.Flags().StringP("user", "u", "", "Username to delete")
	userPasswdCmd.Flags().StringP("user", "u", "", "Username")
	userPasswdCmd.Flags().StringP("pass", "p", "", "New password")
}
