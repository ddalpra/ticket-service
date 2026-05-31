package apperrors

import (
	"errors"
	"net/http"
)

// AppError è un errore applicativo con codice HTTP e messaggio pubblico.
type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

// Costruttori

func NotFound(msg string) *AppError {
	return &AppError{Code: http.StatusNotFound, Message: msg}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: http.StatusUnauthorized, Message: msg}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: http.StatusForbidden, Message: msg}
}

func BadRequest(msg string) *AppError {
	return &AppError{Code: http.StatusBadRequest, Message: msg}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: http.StatusConflict, Message: msg}
}

func Internal(err error) *AppError {
	return &AppError{Code: http.StatusInternalServerError, Message: "errore interno del server", Err: err}
}

// As estrae un AppError dalla catena di errori.
func As(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
