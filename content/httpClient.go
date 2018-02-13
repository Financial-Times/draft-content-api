package content

import (
	"context"
	"io"
	"net/http"
	"strings"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/Financial-Times/service-status-go/buildinfo"
)

func newHttpRequest(ctx context.Context, method string, url string, payload io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, url, payload)
	if err == nil {
		tid, tiderr := tidutils.GetTransactionIDFromContext(ctx)
		if tiderr == nil {
			req.Header.Set(tidutils.TransactionIDHeader, tid)
		}

		req.Header.Set("User-Agent", "PAC-draft-content-api/" + strings.Replace(buildinfo.GetBuildInfo().Version, " ", "-", -1))
	}
	return
}
