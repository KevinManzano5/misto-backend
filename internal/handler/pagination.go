// Package handler contains health handlers
package handler

import (
	"net/http"
	"strconv"
)

// PaginationParams represents pagination params
type PaginationParams struct {
	Cursor string
	Limit  int
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

// ParsePagination function reads cursor and limit in query string
func ParsePagination(request *http.Request) PaginationParams {
	cursor := request.URL.Query().Get("cursor")
	limit := defaultLimit

	if limitQuery := request.URL.Query().Get("limit"); limitQuery != "" {
		if parsed, err := strconv.Atoi(limitQuery); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	return PaginationParams{
		Cursor: cursor,
		Limit:  limit,
	}
}

// PaginatedResponse struct standard format of every paginated list
type PaginatedResponse[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}
