package content

import (
	tIDUtils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type Handler struct {
	endpoint   string
	apyKey     string
	httpClient *http.Client
}

func NewHandler(endpoint string, apiKey string) *Handler {
	return &Handler{endpoint, apiKey, &http.Client{}}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	tID := tIDUtils.GetTransactionIDFromRequest(r)

	cAPIReq, err := http.NewRequest("GET", h.endpoint+"/"+uuid+"?apiKey="+h.apyKey, nil)

	if err != nil {
		log.WithError(err).WithField(tIDUtils.TransactionIDKey, tID).WithField("url", h.endpoint+"/"+uuid).Error("Error in creating the http request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range r.Header {
		cAPIReq.Header[k] = v
	}

	log.WithField(tIDUtils.TransactionIDKey, tID).WithField("url", h.endpoint+"/"+uuid).Info("Calling CAPI")
	resp, err := h.httpClient.Do(cAPIReq)
	if err != nil {
		log.WithError(err).WithField(tIDUtils.TransactionIDKey, tID).WithField("url", h.endpoint+"/"+uuid).Error("Error in calling Content API")
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
