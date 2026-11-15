package repository

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"utils-server/internal/database"
	"utils-server/internal/models"
)

func newTestRepository(t *testing.T) *Repository {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "users.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})

	if err := db.EnsureSchema(); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	return New(db)
}

func TestAddAndGetByID(t *testing.T) {
	repo := newTestRepository(t)

	id, err := repo.Add("alice", []string{"vip", "vip"})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if id <= 0 {
		t.Fatalf("Add() id = %d, want positive", id)
	}

	user, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if user == nil {
		t.Fatal("GetByID() returned nil user")
	}
	if user.ID != id || user.Name != "alice" {
		t.Fatalf("GetByID() = %#v", user)
	}
	if want := []string{"vip", "new"}; !reflect.DeepEqual(user.Tags, want) {
		t.Fatalf("Tags = %v, want %v", user.Tags, want)
	}
}

func TestAddNormalizesNameAndAddsDefaultTag(t *testing.T) {
	repo := newTestRepository(t)

	id, err := repo.Add("  bob  ", nil)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	user, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if user.Name != "bob" {
		t.Fatalf("Name = %q, want bob", user.Name)
	}
	if want := []string{"new"}; !reflect.DeepEqual(user.Tags, want) {
		t.Fatalf("Tags = %v, want %v", user.Tags, want)
	}
}

func TestAddDuplicateName(t *testing.T) {
	repo := newTestRepository(t)

	if _, err := repo.Add("alice", nil); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}

	_, err := repo.Add("  alice  ", nil)
	if !errors.Is(err, models.ErrUserExists) {
		t.Fatalf("second Add() error = %v, want %v", err, models.ErrUserExists)
	}
}

func TestAddInvalidInput(t *testing.T) {
	repo := newTestRepository(t)

	tests := []struct {
		name string
		user string
		tags []string
	}{
		{name: "empty name", user: ""},
		{name: "spaces only name", user: "   "},
		{name: "empty tag", user: "alice", tags: []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := repo.Add(tt.user, tt.tags); !errors.Is(err, models.ErrInvalidInput) {
				t.Fatalf("Add() error = %v, want %v", err, models.ErrInvalidInput)
			}
		})
	}
}

func TestGetByIDNotFound(t *testing.T) {
	repo := newTestRepository(t)

	user, err := repo.GetByID(999)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if user != nil {
		t.Fatalf("GetByID() = %#v, want nil", user)
	}
}

func TestPasswordHash(t *testing.T) {
	repo := newTestRepository(t)

	id, err := repo.Add("alice", nil)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	updated, err := repo.SetPasswordHash(id, "hash-value")
	if err != nil {
		t.Fatalf("SetPasswordHash() error = %v", err)
	}
	if !updated {
		t.Fatal("SetPasswordHash() = false, want true")
	}

	hash, exists, err := repo.GetPasswordHash(id)
	if err != nil {
		t.Fatalf("GetPasswordHash() error = %v", err)
	}
	if !exists || hash != "hash-value" {
		t.Fatalf("GetPasswordHash() = (%q, %v), want (hash-value, true)", hash, exists)
	}

	updated, err = repo.SetPasswordHash(999, "other")
	if err != nil {
		t.Fatalf("SetPasswordHash(missing) error = %v", err)
	}
	if updated {
		t.Fatal("SetPasswordHash(missing) = true, want false")
	}

	hash, exists, err = repo.GetPasswordHash(999)
	if err != nil {
		t.Fatalf("GetPasswordHash(missing) error = %v", err)
	}
	if exists || hash != "" {
		t.Fatalf("GetPasswordHash(missing) = (%q, %v), want (\"\", false)", hash, exists)
	}
}
