package handlers

import "net/http"

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /adduser", s.addUser)
	mux.HandleFunc("GET /adduser", s.methodNotAllowed)
	mux.HandleFunc("PUT /adduser", s.methodNotAllowed)
	mux.HandleFunc("DELETE /adduser", s.methodNotAllowed)
	mux.HandleFunc("PATCH /adduser", s.methodNotAllowed)

	mux.HandleFunc("GET /user/{id}", s.getUser)
	mux.HandleFunc("POST /user/{id}", s.methodNotAllowed)
	mux.HandleFunc("PUT /user/{id}", s.methodNotAllowed)
	mux.HandleFunc("DELETE /user/{id}", s.methodNotAllowed)
	mux.HandleFunc("PATCH /user/{id}", s.methodNotAllowed)

	mux.HandleFunc("PUT /user/{id}/password", s.setPassword)
	mux.HandleFunc("GET /user/{id}/password", s.methodNotAllowed)
	mux.HandleFunc("POST /user/{id}/password", s.methodNotAllowed)
	mux.HandleFunc("DELETE /user/{id}/password", s.methodNotAllowed)
	mux.HandleFunc("PATCH /user/{id}/password", s.methodNotAllowed)

	mux.HandleFunc("POST /user/{id}/password/verify", s.verifyPassword)
	mux.HandleFunc("GET /user/{id}/password/verify", s.methodNotAllowed)
	mux.HandleFunc("PUT /user/{id}/password/verify", s.methodNotAllowed)
	mux.HandleFunc("DELETE /user/{id}/password/verify", s.methodNotAllowed)
	mux.HandleFunc("PATCH /user/{id}/password/verify", s.methodNotAllowed)

	mux.HandleFunc("GET /active", s.getActiveUsers)
	mux.HandleFunc("POST /active", s.methodNotAllowed)
	mux.HandleFunc("PUT /active", s.methodNotAllowed)
	mux.HandleFunc("DELETE /active", s.methodNotAllowed)
	mux.HandleFunc("PATCH /active", s.methodNotAllowed)

	mux.HandleFunc("GET /user", s.methodNotAllowed)
	mux.HandleFunc("POST /user", s.methodNotAllowed)
	mux.HandleFunc("PUT /user", s.methodNotAllowed)
	mux.HandleFunc("DELETE /user", s.methodNotAllowed)
	mux.HandleFunc("PATCH /user", s.methodNotAllowed)

	mux.HandleFunc("/", s.notFound)

	return mux
}
