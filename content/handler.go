package content

import (
	"context"
	"fmt"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/husobee/vestigo"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type Handler struct {
	contentAPI ContentAPI
}

func NewHandler(api ContentAPI) *Handler {
	return &Handler{api}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uuid := vestigo.Param(r, "uuid")
	tID := tidutils.GetTransactionIDFromRequest(r)
	ctx := tidutils.TransactionAwareContext(context.Background(), tID)
	resp, err := h.contentAPI.Get(ctx, uuid)
	if err != nil {
		log.WithError(err).WithField(tidutils.TransactionIDKey, tID).WithField("uuid", uuid).Error("Error in calling Content API")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode < 500 {
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	} else {
		writeErrorMsg(w, "Service unavailable", http.StatusServiceUnavailable)
	}
}

func writeErrorMsg(w http.ResponseWriter, errMsg string, status int) {
	w.WriteHeader(status)
	jsonMsg := fmt.Sprintf(`{"message": "%v"}`, errMsg)
	w.Write([]byte(jsonMsg))
}
