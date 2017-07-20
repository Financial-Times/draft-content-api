package content

import (
	tIDUtils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"io"
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
	// TODO FIX HEADERS

	//cAPIReq.Header = r.Header
	//cAPIReq.Header.Add(tIDUtils.TransactionIDHeader, tID)

	if err != nil {
		log.WithError(err).WithField(tIDUtils.TransactionIDKey, tID).WithField("url", h.endpoint+"/"+uuid).Error("Error in creating the http request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.WithField(tIDUtils.TransactionIDKey, tID).WithField("url", h.endpoint+"/"+uuid).Info("Calling CAPI")
	resp, err := h.httpClient.Do(cAPIReq)
	defer resp.Body.Close()
	if err != nil {
		log.WithError(err).WithField(tIDUtils.TransactionIDKey, tID).WithField("url", h.endpoint+"/"+uuid).Error("Error in calling Content API")
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	io.Copy(w, resp.Body)
}
