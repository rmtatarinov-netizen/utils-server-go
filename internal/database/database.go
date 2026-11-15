package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	mu          sync.Mutex
	initialized bool
}

func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	sqlDB, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	return &DB{DB: sqlDB}, nil
}

func (d *DB) EnsureSchema() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.initialized {
		return nil
	}

	if _, err := d.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		return err
	}

	_, err := d.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			tags TEXT NOT NULL DEFAULT '[]',
			password_hash TEXT
		)
	`)
	if err != nil {
		return err
	}

	var count int
	err = d.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'password_hash'`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		if _, err := d.Exec(`ALTER TABLE users ADD COLUMN password_hash TEXT`); err != nil {
			return err
		}
	}

	d.initialized = true
	return nil
}
