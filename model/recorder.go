package model

import (
	"net/http"
)

type StatusRecorder struct {
	http.ResponseWriter
	Status  int
	Message string
}

func (t *StatusRecorder) Write(byte []byte) (int, error) {
	t.Message = string(byte)
	return t.ResponseWriter.Write(byte)
}

func (t *StatusRecorder) WriteHeader(status int) {
	t.Status = status
	t.ResponseWriter.WriteHeader(status)
}
