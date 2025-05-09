package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/caasmo/restinpieces"
	"github.com/caasmo/restinpieces/core"
	"github.com/caasmo/restinpieces-sqlite-crawshaw"
	"github.com/caasmo/restinpieces-httprouter"
	phuslog "github.com/phuslu/log"
)

// DefaultLoggerOptions provides default settings for slog handlers.
// Level: Debug, Removes time and level attributes from output.
var DefaultLoggerOptions = &slog.HandlerOptions{
	Level: slog.LevelDebug,
	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		//if a.Key == slog.TimeKey || a.Key == slog.LevelKey {
		if a.Key == slog.TimeKey {
			return slog.Attr{} // Return empty Attr to remove
		}
		return a
	},
}

// WithPhusLog configures slog with phuslu/log's JSON handler.
// Uses DefaultLoggerOptions if opts is nil.
func WithPhusLogger(opts *slog.HandlerOptions) core.Option {
	if opts == nil {
		opts = DefaultLoggerOptions // Use package-level defaults
	}
	logger := slog.New(phuslog.SlogNewJSONHandler(os.Stderr, opts))

	// TODO remove slog.SetDefault call? It affects global state.
	slog.SetDefault(logger)
	return core.WithLogger(logger)
}


func main() {
	dbPath := flag.String("db", "", "Path to the SQLite database file (required)")
	ageKeyPath := flag.String("age-key", "", "Path to the age identity (private key) file (required)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -db <database-path> -age-key <identity-file-path>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Start the restinpieces application server.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *dbPath == "" || *ageKeyPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// --- Create the Database Pool ---
	dbPool, err := sqlitecrawshaw.NewCrawshawPool(*dbPath)
	if err != nil {
		slog.Error("failed to create database pool", "error", err)
		os.Exit(1) // Exit if pool creation fails
	}

	// Defer closing the pool here, as main owns it now.
	// This must happen *after* the server finishes.
	defer func() {
		slog.Info("Closing database pool...")
		if err := dbPool.Close(); err != nil {
			slog.Error("Error closing database pool", "error", err)
		}
	}()

	// --- Initialize the Application ---
	_, srv, err := restinpieces.New(
		core.WithAgeKeyPath(*ageKeyPath),
		sqlitecrawshaw.WithDbCrawshaw(dbPool),
		httprouter.WithRouterHttprouter(),
		WithPhusLogger(nil),
	)

	if err != nil {
		slog.Error("failed to initialize application", "error", err)
		// Pool will be closed by the deferred function
		os.Exit(1) // Exit if app initialization fails
	}

	// Start the server
	// The Run method blocks until the server stops (e.g., via signal)
	srv.Run()

	slog.Info("Server shut down gracefully.")
}

