package content

import (
	"context"
	"io"
)

type DraftContentValidator interface {
	Validate(ctx context.Context, contentUUID string, nativeBody io.Reader, contentType string) (io.ReadCloser, error)
	GTG() error
	Endpoint() string
}

type ValidatorError struct {
	httpStatus int
	msg        string
}

func (e ValidatorError) Error() string {
	return e.msg
}

func (e ValidatorError) StatusCode() int {
	return e.httpStatus
}
