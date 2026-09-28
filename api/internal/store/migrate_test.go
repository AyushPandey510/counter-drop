package store

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"counter-drop/api/internal/domain"

	"github.com/jackc/pgx/v5"
)

// Two API containers starting together must not both apply the same migration.
func TestMigrationsConcurrentStartup(t *testing.T) {
	dsn := os.Getenv("CD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CD_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	// Own schema, so this can run alongside the API tests that reset "public".
	if _, err := conn.Exec(ctx, `DROP SCHEMA IF EXISTS cd_migrate_test CASCADE; CREATE SCHEMA cd_migrate_test;`); err != nil {
		t.Fatal(err)
	}
	conn.Close(ctx)
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	dsn += sep + "search_path=cd_migrate_test"
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", "migrations")

	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			st, err := New(ctx, dsn, domain.DefaultPolicy())
			if err != nil {
				errs[i] = err
				return
			}
			defer st.Close()
			errs[i] = st.ApplyMigrations(ctx, dir)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("instance %d: %v", i, err)
		}
	}
}
