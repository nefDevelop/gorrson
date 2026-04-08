package cmd

import (
	"crypto/md5"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new user and database",
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

		if err := store.UpsertUser(user, hash, feverBcrypt); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("User %s registered successfully.\n", user)
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)
	registerCmd.Flags().StringP("user", "n", "", "Username")
	registerCmd.Flags().StringP("pass", "p", "", "Password")
}
