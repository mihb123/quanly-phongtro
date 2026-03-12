package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func Connect(dataSourceName string) (*sql.DB, error) {
	databaseConnection, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}

	databaseConnection.SetMaxOpenConns(25)
	databaseConnection.SetMaxIdleConns(25)
	databaseConnection.SetConnMaxLifetime(30 * time.Minute)
	databaseConnection.SetConnMaxIdleTime(5 * time.Minute)

	connectionContext, cancelConnectionTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelConnectionTimeout()
	if err := databaseConnection.PingContext(connectionContext); err != nil {
		_ = databaseConnection.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	return databaseConnection, nil
}

func Disconnect(_ context.Context, databaseConnection *sql.DB) {
	if databaseConnection != nil {
		_ = databaseConnection.Close()
	}
}
