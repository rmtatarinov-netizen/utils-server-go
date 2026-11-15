package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"utils-server/internal/database"
	"utils-server/internal/models"
	"utils-server/internal/repository"
	"utils-server/internal/services/password"
	"utils-server/internal/services/presence"
)

func newTestMux(t *testing.T, limit int) *http.ServeMux {
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

	activeUsers, err := presence.New(limit)
	if err != nil {
		t.Fatalf("presence.New() error = %v", err)
	}

	server := New(repository.New(db), password.New(), activeUsers)
	return server.Routes()
}

func perform(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func createUser(t *testing.T, handler http.Handler, name string) int {
	t.Helper()

	rec := perform(t, handler, http.MethodPost, "/adduser", fmt.Sprintf(`{"name":%q}`, name))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /adduser status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var response struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode id response: %v", err)
	}
	return response.ID
}

func TestAddUserAndGetUser(t *testing.T) {
	handler := newTestMux(t, 10)

	rec := perform(t, handler, http.MethodPost, "/adduser", `{"name":"  alice  ","tags":["vip"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /adduser status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID != 1 {
		t.Fatalf("created id = %d, want 1", created.ID)
	}

	rec = perform(t, handler, http.MethodGet, "/user/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /user/1 status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "password_hash") {
		t.Fatalf("GET /user/1 leaks password_hash: %s", rec.Body.String())
	}

	var user models.User
	if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
		t.Fatalf("decode user response: %v", err)
	}
	if user.ID != 1 || user.Name != "alice" {
		t.Fatalf("user = %#v", user)
	}
	if len(user.Tags) != 2 || user.Tags[0] != "vip" || user.Tags[1] != "new" {
		t.Fatalf("tags = %v, want [vip new]", user.Tags)
	}
}

func TestAddUserValidation(t *testing.T) {
	handler := newTestMux(t, 10)

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "not an object", body: `[]`},
		{name: "empty body", body: ""},
		{name: "empty name", body: `{"name":""}`},
		{name: "spaces name", body: `{"name":"   "}`},
		{name: "non-string name", body: `{"name":123}`},
		{name: "non-array tags", body: `{"name":"alice","tags":"vip"}`},
		{name: "non-string tag", body: `{"name":"alice","tags":[1]}`},
		{name: "trailing data", body: `{"name":"alice"}{"name":"bob"}`},
		{name: "too large", body: `{"name":"` + strings.Repeat("a", maxRequestBodyBytes) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := perform(t, handler, http.MethodPost, "/adduser", tt.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAddUserDuplicate(t *testing.T) {
	handler := newTestMux(t, 10)

	createUser(t, handler, "alice")

	rec := perform(t, handler, http.MethodPost, "/adduser", `{"name":"alice"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetUserNotFound(t *testing.T) {
	handler := newTestMux(t, 10)

	for _, path := range []string{"/user/999", "/user/abc", "/user/0", "/user/-1"} {
		t.Run(path, func(t *testing.T) {
			rec := perform(t, handler, http.MethodGet, path, "")
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	handler := newTestMux(t, 10)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/adduser"},
		{method: http.MethodDelete, path: "/user/1"},
		{method: http.MethodGet, path: "/user/1/password"},
		{method: http.MethodDelete, path: "/user/1/password"},
		{method: http.MethodGet, path: "/user/1/password/verify"},
		{method: http.MethodPut, path: "/user/1/password/verify"},
		{method: http.MethodPost, path: "/active"},
		{method: http.MethodDelete, path: "/active"},
		{method: http.MethodGet, path: "/user"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := perform(t, handler, tt.method, tt.path, "")
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want 405; body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestUnknownRoute(t *testing.T) {
	handler := newTestMux(t, 10)

	rec := perform(t, handler, http.MethodGet, "/unknown", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
}

func TestSetPasswordAndVerify(t *testing.T) {
	handler := newTestMux(t, 10)
	id := createUser(t, handler, "alice")

	rec := perform(t, handler, http.MethodPut, fmt.Sprintf("/user/%d/password", id), `{"password":"s3cret"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT password status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("PUT password body = %q, want empty", rec.Body.String())
	}

	assertValid := func(t *testing.T, password string, want bool) {
		t.Helper()
		rec := perform(t, handler, http.MethodPost, fmt.Sprintf("/user/%d/password/verify", id), fmt.Sprintf(`{"password":%q}`, password))
		if rec.Code != http.StatusOK {
			t.Fatalf("verify status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var response struct {
			Valid bool `json:"valid"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode verify response: %v", err)
		}
		if response.Valid != want {
			t.Fatalf("valid = %v, want %v", response.Valid, want)
		}
	}

	assertValid(t, "s3cret", true)
	assertValid(t, "wrong", false)

	rec = perform(t, handler, http.MethodGet, fmt.Sprintf("/user/%d", id), "")
	if strings.Contains(rec.Body.String(), "s3cret") {
		t.Fatalf("GET user leaks password: %s", rec.Body.String())
	}
}

func TestVerifyPasswordWithoutStoredHash(t *testing.T) {
	handler := newTestMux(t, 10)
	id := createUser(t, handler, "alice")

	rec := perform(t, handler, http.MethodPost, fmt.Sprintf("/user/%d/password/verify", id), `{"password":"s3cret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Valid {
		t.Fatal("valid = true for user without password hash")
	}
}

func TestPasswordValidation(t *testing.T) {
	handler := newTestMux(t, 10)
	id := createUser(t, handler, "alice")

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{name: "set missing user", method: http.MethodPut, path: "/user/999/password", body: `{"password":"s3cret"}`, want: http.StatusNotFound},
		{name: "set invalid id", method: http.MethodPut, path: "/user/abc/password", body: `{"password":"s3cret"}`, want: http.StatusNotFound},
		{name: "verify missing user", method: http.MethodPost, path: "/user/999/password/verify", body: `{"password":"s3cret"}`, want: http.StatusNotFound},
		{name: "verify invalid id", method: http.MethodPost, path: "/user/abc/password/verify", body: `{"password":"s3cret"}`, want: http.StatusNotFound},
		{name: "set missing field", method: http.MethodPut, path: fmt.Sprintf("/user/%d/password", id), body: `{}`, want: http.StatusBadRequest},
		{name: "set empty password", method: http.MethodPut, path: fmt.Sprintf("/user/%d/password", id), body: `{"password":""}`, want: http.StatusBadRequest},
		{name: "set too long password", method: http.MethodPut, path: fmt.Sprintf("/user/%d/password", id), body: `{"password":"` + strings.Repeat("a", maxPasswordLength+1) + `"}`, want: http.StatusBadRequest},
		{name: "set invalid json", method: http.MethodPut, path: fmt.Sprintf("/user/%d/password", id), body: `{`, want: http.StatusBadRequest},
		{name: "verify missing field", method: http.MethodPost, path: fmt.Sprintf("/user/%d/password/verify", id), body: `{}`, want: http.StatusBadRequest},
		{name: "verify empty password", method: http.MethodPost, path: fmt.Sprintf("/user/%d/password/verify", id), body: `{"password":""}`, want: http.StatusBadRequest},
		{name: "verify invalid json", method: http.MethodPost, path: fmt.Sprintf("/user/%d/password/verify", id), body: `{`, want: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := perform(t, handler, tt.method, tt.path, tt.body)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

func TestActiveUsers(t *testing.T) {
	handler := newTestMux(t, 2)

	rec := perform(t, handler, http.MethodGet, "/active", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /active status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var initial struct {
		ActiveUsers []int `json:"active_users"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode active response: %v", err)
	}
	if initial.ActiveUsers == nil || len(initial.ActiveUsers) != 0 {
		t.Fatalf("active_users = %#v, want empty slice", initial.ActiveUsers)
	}

	createUser(t, handler, "alice")
	createUser(t, handler, "bob")
	createUser(t, handler, "carol")

	rec = perform(t, handler, http.MethodGet, "/active", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode active response: %v", err)
	}
	if want := []int{2, 3}; len(initial.ActiveUsers) != 2 || initial.ActiveUsers[0] != want[0] || initial.ActiveUsers[1] != want[1] {
		t.Fatalf("active_users = %v, want %v", initial.ActiveUsers, want)
	}

	rec = perform(t, handler, http.MethodGet, "/user/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /user/1 status = %d", rec.Code)
	}

	rec = perform(t, handler, http.MethodGet, "/active", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode active response: %v", err)
	}
	if want := []int{3, 1}; len(initial.ActiveUsers) != 2 || initial.ActiveUsers[0] != want[0] || initial.ActiveUsers[1] != want[1] {
		t.Fatalf("active_users = %v, want %v", initial.ActiveUsers, want)
	}

	rec = perform(t, handler, http.MethodGet, "/user/3", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /user/3 status = %d", rec.Code)
	}

	rec = perform(t, handler, http.MethodGet, "/active", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode active response: %v", err)
	}
	if want := []int{1, 3}; len(initial.ActiveUsers) != 2 || initial.ActiveUsers[0] != want[0] || initial.ActiveUsers[1] != want[1] {
		t.Fatalf("active_users = %v, want %v", initial.ActiveUsers, want)
	}
}
