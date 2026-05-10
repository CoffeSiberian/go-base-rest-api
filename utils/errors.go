package utils

import "net/http"

type AppError struct {
	HTTPCode int
	Code     string
	Message  string
}

func (e *AppError) Error() string { return e.Message }

var (
	ErrNotFound      = &AppError{http.StatusNotFound, "NOT_FOUND", "Resource not found"}
	ErrUnauthorized  = &AppError{http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required"}
	ErrForbidden     = &AppError{http.StatusForbidden, "FORBIDDEN", "Insufficient permissions"}
	ErrBadRequest    = &AppError{http.StatusBadRequest, "BAD_REQUEST", "Invalid request"}
	ErrInternal      = &AppError{http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error"}
)
