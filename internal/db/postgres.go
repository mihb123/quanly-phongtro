package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func NewPostgres(dsn string) (*bun.DB, error) {
	sqldb, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqldb.SetMaxOpenConns(20)
	sqldb.SetMaxIdleConns(10)
	sqldb.SetConnMaxLifetime(30 * time.Minute)

	if err := sqldb.Ping(); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	db := bun.NewDB(sqldb, pgdialect.New())
	return db, nil
}
