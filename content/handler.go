package content

import (
	"context"
	tIDUtils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type Handler struct {
	cAPI *ContentAPI
}

func NewHandler(cAPI *ContentAPI) *Handler {
	return &Handler{cAPI}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	tID := tIDUtils.GetTransactionIDFromRequest(r)
	ctx := tIDUtils.TransactionAwareContext(context.Background(), tID)
	resp, err := h.cAPI.get(ctx, uuid, r.Header)
	if err != nil {
		log.WithError(err).WithField(tIDUtils.TransactionIDKey, tID).WithField("url", h.cAPI.endpoint+"/"+uuid).Error("Error in calling Content API")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
