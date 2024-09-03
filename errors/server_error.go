package errors

import (
	"fmt"
	"log/slog"
)

type ServerError struct {
	Message string
	Err     error
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("client error (%v)", e.Err)
}

func (e *ServerError) Unwrap() error {
	return e.Err
}

func (e *ServerError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", "client"),
		slog.String("message", e.Err.Error()),
	)
}

type MeshAction uint

const (
	PassThrough MeshAction = iota
)
