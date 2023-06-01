package repository

import "errors"

var (
	ErrAlreadyExists = errors.New("repository: already exists")
	ErrNotFound      = errors.New("repository: not found")
)
