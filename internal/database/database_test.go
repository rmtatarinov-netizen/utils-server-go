package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func openTestDB(t *testing.T) (*DB, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "nested", "users.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	return db, path
}

func TestOpenCreatesDirectory(t *testing.T) {
	_, path := openTestDB(t)

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("database directory does not exist: %v", err)
	}
}

func TestEnsureSchemaIsIdempotent(t *testing.T) {
	db, _ := openTestDB(t)

	if err := db.EnsureSchema(); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
	if err := db.EnsureSchema(); err != nil {
		t.Fatalf("second EnsureSchema() error = %v", err)
	}

	var columns int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users')`).Scan(&columns)
	if err != nil {
		t.Fatalf("count columns error = %v", err)
	}
	if columns != 4 {
		t.Fatalf("users table has %d columns, want 4", columns)
	}

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'password_hash'`).Scan(&count)
	if err != nil {
		t.Fatalf("count password_hash error = %v", err)
	}
	if count != 1 {
		t.Fatalf("password_hash column count = %d, want 1", count)
	}

	var users int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		t.Fatalf("count users error = %v", err)
	}
	if users != 0 {
		t.Fatalf("users count = %d, want 0", users)
	}
}

func TestEnsureSchemaEnablesWAL(t *testing.T) {
	db, _ := openTestDB(t)

	if err := db.EnsureSchema(); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("journal_mode error = %v", err)
	}
	if !strings.EqualFold(mode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}
