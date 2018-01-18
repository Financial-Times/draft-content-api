package content

import (
	"net/http"
)

type DraftContentMapper interface {
	GTG() error
	Endpoint() string
}

type draftContentMapper struct {
	pacExternalService
}

func NewDraftContentMapperService(endpoint string) DraftContentMapper {
	return &draftContentMapper{pacExternalService{endpoint, &http.Client{}}}
}
