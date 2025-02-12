package main

import (
	"flag"
	"fmt"
	"log/slog"
	"openheart.tylery.com/internal/env"
	"os"
	"runtime/debug"
	"sync"

	"openheart.tylery.com/internal/database"
	"openheart.tylery.com/internal/version"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	err := run(logger)
	if err != nil {
		trace := string(debug.Stack())
		logger.Error(err.Error(), "trace", trace)
		os.Exit(1)
	}
}

type config struct {
	httpPort int
	db       struct {
		dsn string
	}
}

type application struct {
	config config
	db     *database.DB
	logger *slog.Logger
	wg     sync.WaitGroup
}

func run(logger *slog.Logger) error {
	var cfg config

	// The intent for config here is to allow environment variables, however if they do it inline it is overridden
	// by inline flags
	var httpPort int
	var dsn string

	flag.IntVar(&httpPort, "http-port", 0, "Default 4444")
	flag.StringVar(&dsn, "dsn", "", "Database DSN (default user:password@tcp(host:port)/database)")

	if httpPort != 0 {
		cfg.httpPort = httpPort
	} else {
		cfg.httpPort = env.GetInt("HTTP_PORT", 4444)
	}

	if dsn != "" {
		cfg.db.dsn = dsn
	} else {
		cfg.db.dsn = env.GetString("DB_DSN", "user:pass@localhost:3306/db")
	}

	showVersion := flag.Bool("version", false, "display version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\n", version.Get())
		return nil
	}

	db, err := database.New(cfg.db.dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	app := application{
		config: cfg,
		db:     db,
		logger: logger,
	}

	return app.serveHTTP()
}
