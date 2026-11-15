package handlers

import (
	"utils-server/internal/repository"
	"utils-server/internal/services/password"
	"utils-server/internal/services/presence"
)

type Server struct {
	repo        *repository.Repository
	passwordSvc *password.Service
	activeUsers *presence.ActiveUsers
}

func New(repo *repository.Repository, passwordSvc *password.Service, activeUsers *presence.ActiveUsers) *Server {
	return &Server{
		repo:        repo,
		passwordSvc: passwordSvc,
		activeUsers: activeUsers,
	}
}
