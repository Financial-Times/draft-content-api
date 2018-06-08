package content

import (
	"context"
	"io"
)

type DraftContentMapper interface {
	MapNativeContent(ctx context.Context, contentUUID string, nativeBody io.Reader, contentType string) (io.ReadCloser, error)
	GTG() error
	Endpoint() string
}

type MapperError struct {
	httpStatus int
	msg        string
}

func (e MapperError) Error() string {
	return e.msg
}

func (e MapperError) MapperStatusCode() int {
	return e.httpStatus
}
