package model

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict error")
	ErrInternal     = errors.New("internal error")
	ErrInvalid      = errors.New("invalid data")
	ErrUnauthorized = errors.New("unauthorized")
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
