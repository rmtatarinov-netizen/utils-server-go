package models

import "errors"

var (
	ErrUserExists   = errors.New("user already exists")
	ErrInvalidInput = errors.New("invalid input")
)
