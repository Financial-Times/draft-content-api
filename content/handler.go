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
	contentAPI *ContentAPI
}

func NewHandler(api *ContentAPI) *Handler {
	return &Handler{api}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	tID := tIDUtils.GetTransactionIDFromRequest(r)
	ctx := tIDUtils.TransactionAwareContext(context.Background(), tID)
	resp, err := h.contentAPI.get(ctx, uuid, r.Header)
	if err != nil {
		log.WithError(err).WithField(tIDUtils.TransactionIDKey, tID).WithField("url", h.contentAPI.endpoint+"/"+uuid).Error("Error in calling Content API")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	io.Copy(w, resp.Body)
	return
}
