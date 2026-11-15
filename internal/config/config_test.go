package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettingsDefaults(t *testing.T) {
	t.Setenv("DB_PATH", "")
	t.Setenv("ACTIVE_USERS_LIMIT", "")
	t.Setenv("PORT", "")

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if settings.Port != defaultPort {
		t.Fatalf("Port = %q, want %q", settings.Port, defaultPort)
	}
	if settings.ActiveUsersLimit != defaultActiveUsersLimit {
		t.Fatalf("ActiveUsersLimit = %d, want %d", settings.ActiveUsersLimit, defaultActiveUsersLimit)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	wantPath := filepath.Join(wd, defaultDBPath)
	if settings.DBPath != wantPath {
		t.Fatalf("DBPath = %q, want %q", settings.DBPath, wantPath)
	}
}

func TestLoadSettingsFromEnv(t *testing.T) {
	t.Setenv("DB_PATH", "/tmp/custom.db")
	t.Setenv("ACTIVE_USERS_LIMIT", "10")
	t.Setenv("PORT", "9090")

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if settings.DBPath != "/tmp/custom.db" {
		t.Fatalf("DBPath = %q", settings.DBPath)
	}
	if settings.ActiveUsersLimit != 10 {
		t.Fatalf("ActiveUsersLimit = %d, want 10", settings.ActiveUsersLimit)
	}
	if settings.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", settings.Port)
	}
}

func TestLoadSettingsRejectsInvalidLimit(t *testing.T) {
	for _, value := range []string{"0", "-1", "abc"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("DB_PATH", "")
			t.Setenv("ACTIVE_USERS_LIMIT", value)
			t.Setenv("PORT", "")

			if _, err := LoadSettings(); err == nil {
				t.Fatalf("LoadSettings() with ACTIVE_USERS_LIMIT=%q returned nil error", value)
			}
		})
	}
}
