package http

//go:generate cmtstringer -type StatusCode

// StatusCode type of HTTP status code constant
type StatusCode int

const (
	// StatusBadRequest Bad Request
	StatusBadRequest StatusCode = 400
	// StatusNotFound Not Found
	StatusNotFound StatusCode = 404
)
