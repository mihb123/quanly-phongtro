// Package repotest provides shared test helpers for repository package tests.
package repotest

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// SetupTestDB creates a bun.DB backed by a sqlmock stub connection for repository tests.
func SetupTestDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	bunDB := bun.NewDB(db, pgdialect.New())
	return bunDB, mock
}
