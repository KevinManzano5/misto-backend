// Package domain contains rules business
package domain

import "errors"

var (
	// ErrNotFound reutilize
	ErrNotFound = errors.New("resource not found")
	// ErrAlreadyExists reutilize
	ErrAlreadyExists = errors.New("resource already exists")
	// ErrUnauthorized reutilize
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden reutilize
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidInput reutilize
	ErrInvalidInput = errors.New("invalid input")
	// ErrInternal reutilize
	ErrInternal = errors.New("internal error")
)
