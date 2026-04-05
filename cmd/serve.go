package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
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
		logFormat, _ := cmd.Flags().GetString("log-format")
		
		if port == "" {
			port = os.Getenv("GORRSON_PORT")
			if port == "" {
				port = "7077"
			}
		}

		// Ensure data directory exists
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			os.MkdirAll(dataDir, 0755)
		}

		// Setup dual logging (Stdout + File)
		logPath := filepath.Join(dataDir, "gorrson.log")
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		
		var mw io.Writer = os.Stdout
		if err == nil {
			mw = io.MultiWriter(os.Stdout, logFile)
		} else {
			fmt.Printf("Could not open log file %s: %v\n", logPath, err)
		}

		var handler slog.Handler
		opts := &slog.HandlerOptions{Level: slog.LevelInfo}
		if logFormat == "json" {
			handler = slog.NewJSONHandler(mw, opts)
		} else {
			handler = slog.NewTextHandler(mw, opts)
		}
		
		slog.SetDefault(slog.New(handler))

		slog.Info("gorrson server starting", slog.String("port", port), slog.String("format", logFormat))

		// Context for workers and server shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start workers for all existing users found in the data directory
		files, err := os.ReadDir(dataDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".db") && file.Name() != "global.db" {
					username := strings.TrimSuffix(file.Name(), ".db")
					
					store, err := db.NewStore(dataDir, username)
					if err != nil {
						slog.Error("Could not load database", slog.String("user", username), slog.Any("error", err))
						continue
					}
					
					// Each user gets a background worker to sync their feeds independently
					worker := rss.NewWorker(store, 15*time.Minute)
					go worker.Start(ctx)
					slog.Info("Started background worker", slog.String("user", username))
				}
			}
		}

		server := api.NewServer(dataDir)
		addr := ":" + port
		srv := &http.Server{
			Addr:    addr,
			Handler: server.GetRouter(), // Note: Changed to GetRouter() to match common naming
		}

		// Channel to listen for signals
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		go func() {
			slog.Info("listening for requests", slog.String("addr", addr))
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("server fatal error", slog.Any("error", err))
				os.Exit(1)
			}
		}()

		// Wait for signal
		sig := <-stop
		slog.Info("shutting down server", slog.String("signal", sig.String()))

		// Cancel context to stop workers
		cancel()

		// Graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown failed", slog.Any("error", err))
		}
		slog.Info("server exited gracefully")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("port", "p", "7077", "Port to listen on")
	serveCmd.Flags().StringP("log-format", "f", "text", "Log format (text or json)")
}
