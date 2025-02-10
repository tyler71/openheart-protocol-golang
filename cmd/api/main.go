package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"

	"openheart.tylery.com/internal/database"
	"openheart.tylery.com/internal/env"
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
	baseURL  string
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

	cfg.httpPort = env.GetInt("HTTP_PORT", 4444)
	cfg.baseURL = env.GetString("BASE_URL", fmt.Sprintf("http://localhost:%d", cfg.httpPort))
	cfg.db.dsn = env.GetString("DB_DSN", "user:pass@localhost:3306/db")

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
