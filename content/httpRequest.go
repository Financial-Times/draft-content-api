package content

import (
	"context"
	"io"
	"net/http"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
)

func newHttpRequest(ctx context.Context, method string, url string, payload io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, url, payload)
	if err == nil {
		tid, tidErr := tidutils.GetTransactionIDFromContext(ctx)
		if tidErr == nil {
			req.Header.Set(tidutils.TransactionIDHeader, tid)
		}
		req = req.WithContext(ctx)
	}
	return
}
