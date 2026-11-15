package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

const (
	defaultDBPath           = "users.db"
	defaultActiveUsersLimit = 5
	defaultPort             = "8000"
)

type Settings struct {
	DBPath           string
	ActiveUsersLimit int
	Port             string
}

func LoadSettings() (*Settings, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		dbPath = filepath.Join(dir, defaultDBPath)
	}

	limit := defaultActiveUsersLimit
	if v := os.Getenv("ACTIVE_USERS_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return nil, errors.New("ACTIVE_USERS_LIMIT must be a positive integer")
		}
		limit = n
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	return &Settings{
		DBPath:           dbPath,
		ActiveUsersLimit: limit,
		Port:             port,
	}, nil
}
