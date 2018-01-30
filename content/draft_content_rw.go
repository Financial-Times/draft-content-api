package content

import (
	"net/http"
)

type DraftContentRW interface {
	GTG() error
	Endpoint() string
}

type draftContentRW struct {
	pacExternalService
}

func NewDraftContentRWService(endpoint string) DraftContentRW {
	return &draftContentRW{pacExternalService{endpoint, &http.Client{}}}
}
