package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand"
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
		logLevelStr, _ := cmd.Flags().GetString("log-level")
		silent, _ := cmd.Flags().GetBool("silent")
		
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

		// Determine log level
		var level slog.Level
		switch strings.ToUpper(logLevelStr) {
		case "DEBUG":
			level = slog.LevelDebug
		case "WARN":
			level = slog.LevelWarn
		case "ERROR":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}

		// Override with silent flag for backward compatibility
		if silent {
			level = slog.LevelError
		}

		var handler slog.Handler
		opts := &slog.HandlerOptions{Level: level}
		if logFormat == "json" {
			handler = slog.NewJSONHandler(mw, opts)
		} else {
			handler = slog.NewTextHandler(mw, opts)
		}
		
		slog.SetDefault(slog.New(handler))

		// If level is ERROR or higher, discard standard log output
		if level >= slog.LevelError {
			log.SetOutput(io.Discard)
		}

		slog.Info("gorrson server starting", slog.String("port", port), slog.String("format", logFormat), slog.String("level", level.String()))

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
					
					// Use a small jitter for initial sync to avoid "thundering herd"
					go func(w *rss.Worker, user string) {
						// Wait up to 2 minutes to spread the load
						jitter := time.Duration(rand.Intn(120)) * time.Second
						slog.Debug("Jittering initial sync", slog.String("user", user), slog.Duration("wait", jitter))
						time.Sleep(jitter)
						w.Start(ctx)
					}(worker, username)
					slog.Info("Started background worker", slog.String("user", username))
				}
			}
		}

		server := api.NewServer(dataDir, silent)
		addr := ":" + port
		srv := &http.Server{
			Addr:    addr,
			Handler: server.GetRouter(),
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
	serveCmd.Flags().StringP("log-level", "l", "debug", "Min log level (debug, info, warn, error)")
	serveCmd.Flags().BoolP("silent", "s", false, "Deprecated: use --log-level error instead")
}
