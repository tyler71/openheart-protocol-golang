package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jmoiron/sqlx"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const defaultTimeout = 3 * time.Second

type DB struct {
	*sqlx.DB
}

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

	m, err := migrate.New("file://internal/database/migrations", fmt.Sprintf("mysql://%s", dsn))
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
