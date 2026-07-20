// Package dbtest provides a shared PostgreSQL test-database helper used by
// the identity, api, and e2e test suites.
package dbtest

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open は TEST_DATABASE_URL が設定されている場合のみ実PostgreSQLへ接続する。
// 未設定ならDB依存のテストをskipする
// (例: `docker compose up -d` してから `TEST_DATABASE_URL=... go test ./...`)。
func Open(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping test that requires PostgreSQL")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	truncate := func() {
		if _, err := db.Exec(`TRUNCATE TABLE transitions, entities`); err != nil {
			t.Fatal(err)
		}
	}
	truncate()
	t.Cleanup(truncate)

	return db
}
