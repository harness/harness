package cache

import "errors"

var (
	// ErrNotFound holds exported `not found error` for not found items
	ErrNotFound = errors.New("not found")
)
