package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
)

const defaultTimeout = 3 * time.Second

type DB struct {
	*sqlx.DB
}

//go:embed migrations
var migrations embed.FS

func New(dsn string) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	db, err := sqlx.ConnectContext(ctx, "mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(2 * time.Hour)

	sourceInstance, err := httpfs.New(http.FS(migrations), "migrations")
	if err != nil {
		log.Fatal(err)
	}
	m, err := migrate.NewWithSourceInstance("httpfs", sourceInstance, fmt.Sprintf("mysql://%s", dsn))
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			log.Fatal(err)
		}
	}

	return &DB{db}, nil
}
