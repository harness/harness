package parser

import "errors"

var (
	ErrSHADoesNotMatch = errors.New("sha does not match")
	ErrHunkNotFound    = errors.New("hunk not found")
)
