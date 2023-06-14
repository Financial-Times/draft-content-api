package content

import (
	"context"
	"io"

	"github.com/Financial-Times/go-logger/v2"
)

type DraftContentValidator interface {
	Validate(ctx context.Context, contentUUID string, nativeBody io.Reader, contentType string, log *logger.UPPLogger) (io.ReadCloser, error)
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
