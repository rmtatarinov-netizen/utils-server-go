package handlers

import (
	"errors"
	"net/http"

	"utils-server/internal/models"
	"utils-server/internal/utils"
)

type passwordRequest struct {
	Password string `json:"password"`
}

func (s *Server) addUser(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	if err := decodeJSON(w, r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be a JSON object")
		return
	}

	name, err := utils.NormalizeName(payload.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "'name' must be a non-empty string")
		return
	}

	tags, err := utils.NormalizeTags(payload.Tags)
	if err != nil {
		writeError(w, http.StatusBadRequest, "'tags' must be a list of strings")
		return
	}

	id, err := s.repo.Add(name, tags)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrUserExists):
			writeError(w, http.StatusConflict, "user already exists")
		case errors.Is(err, models.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	s.activeUsers.Add(id)

	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	user, err := s.repo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	s.activeUsers.Add(id)

	writeJSON(w, http.StatusOK, user)
}

func (s *Server) setPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var payload passwordRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be a JSON object")
		return
	}

	if payload.Password == "" || len(payload.Password) > maxPasswordLength {
		writeError(w, http.StatusBadRequest, "'password' must be a non-empty string of at most 1024 characters")
		return
	}

	user, err := s.repo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	hash, err := s.passwordSvc.Hash(payload.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	updated, err := s.repo.SetPasswordHash(id, hash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !updated {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) verifyPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var payload passwordRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be a JSON object")
		return
	}

	if payload.Password == "" || len(payload.Password) > maxPasswordLength {
		writeError(w, http.StatusBadRequest, "'password' must be a non-empty string of at most 1024 characters")
		return
	}

	user, err := s.repo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	hash, exists, err := s.repo.GetPasswordHash(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !exists {
		writeJSON(w, http.StatusOK, map[string]bool{"valid": false})
		return
	}

	valid, err := s.passwordSvc.Verify(hash, payload.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"valid": valid})
}

func (s *Server) getActiveUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"active_users": s.activeUsers.Snapshot()})
}

func (s *Server) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}
