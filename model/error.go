package model

import "errors"

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict error")
	ErrInternal = errors.New("internal error")
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
