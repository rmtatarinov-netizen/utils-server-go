package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"

	"utils-server/internal/database"
	"utils-server/internal/models"
	"utils-server/internal/utils"
)

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Add(name string, tags []string) (int, error) {
	name, err := utils.NormalizeName(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", models.ErrInvalidInput, err)
	}

	tags, err = utils.NormalizeTags(tags)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", models.ErrInvalidInput, err)
	}

	seen := make(map[string]struct{}, len(tags)+1)
	cleanTags := make([]string, 0, len(tags)+1)
	for _, tag := range tags {
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		cleanTags = append(cleanTags, tag)
	}
	if _, ok := seen["new"]; !ok {
		cleanTags = append(cleanTags, "new")
	}

	tagsJSON, err := json.Marshal(cleanTags)
	if err != nil {
		return 0, err
	}

	result, err := r.db.Exec(`INSERT INTO users (name, tags) VALUES (?, ?)`, name, string(tagsJSON))
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, models.ErrUserExists
		}
		return 0, err
	}

	id64, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id64), nil
}

func (r *Repository) GetByID(id int) (*models.User, error) {
	row := r.db.QueryRow(`SELECT id, name, tags FROM users WHERE id = ?`, id)
	return r.scanRow(row)
}

func (r *Repository) scanRow(row *sql.Row) (*models.User, error) {
	var id int
	var name string
	var tagsJSON string
	err := row.Scan(&id, &name, &tagsJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var tags []string
	if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
		tags = []string{}
	}

	return &models.User{ID: id, Name: name, Tags: tags}, nil
}

func (r *Repository) SetPasswordHash(id int, hash string) (bool, error) {
	result, err := r.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *Repository) GetPasswordHash(id int) (string, bool, error) {
	var hash sql.NullString
	err := r.db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, id).Scan(&hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	if !hash.Valid {
		return "", false, nil
	}
	return hash.String, true, nil
}
