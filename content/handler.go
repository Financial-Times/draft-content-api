package content

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/husobee/vestigo"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type Handler struct {
	uppContentAPI ContentAPI
	contentRW     DraftContentRW
}

func NewHandler(uppApi ContentAPI, draftContentRW DraftContentRW) *Handler {
	return &Handler{uppApi, draftContentRW}
}

func (h *Handler) ReadContent(w http.ResponseWriter, r *http.Request) {
	uuid := vestigo.Param(r, "uuid")
	tID := tidutils.GetTransactionIDFromRequest(r)
	ctx := tidutils.TransactionAwareContext(context.Background(), tID)

	var status = http.StatusInternalServerError
	content, err := h.contentRW.Read(ctx, uuid)
	if err == nil {
		defer content.Close()
		status = http.StatusOK
	} else {
		if err == ErrDraftNotFound {
			readLog := log.WithField(tidutils.TransactionIDKey, tID).WithField("uuid", uuid)
			readLog.Warn("Draft not found in PAC, trying UPP")
			uppResp, err := h.uppContentAPI.Get(ctx, uuid)
			if err != nil {
				readLog.WithError(err).Error("Error in calling Content API")
				writeMessage(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer uppResp.Body.Close()
			if uppResp.StatusCode == http.StatusOK || uppResp.StatusCode == http.StatusNotFound || uppResp.StatusCode == http.StatusBadRequest {
				status = uppResp.StatusCode
				content = uppResp.Body
			}
		} else if err == ErrDraftNotMappable {
			status = http.StatusUnprocessableEntity
		}

		if content == nil {
			msg := errorMessageForRead(status)
			writeMessage(w, msg, status)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	io.Copy(w, content)
}

func (h *Handler) WriteNativeContent(w http.ResponseWriter, r *http.Request) {
	uuid := vestigo.Param(r, "uuid")
	tID := tidutils.GetTransactionIDFromRequest(r)
	ctx := tidutils.TransactionAwareContext(context.Background(), tID)

	writeLog := log.WithField(tidutils.TransactionIDKey, tID).WithField("uuid", uuid)

	if err := validateUUID(uuid); err != nil {
		writeLog.WithError(err).Error("Invalid content UUID")
		writeMessage(w, fmt.Sprintf("Invalid content UUID: %v", uuid), http.StatusBadRequest)
		return
	}

	originSystemId, err := validateOrigin(r.Header.Get(originSystemIdHeader))
	if err != nil {
		writeLog.WithError(err).Error("Invalid origin system id")
		writeMessage(w, fmt.Sprintf("Invalid origin system id: %v", originSystemId), http.StatusBadRequest)
		return
	}

	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeLog.WithError(err).Error("Unable to read draft content body")
		writeMessage(w, fmt.Sprintf("Unable to read draft content body: %v", err.Error()), http.StatusBadRequest)
		return
	}
	draftContent := string(raw)
	draftHeaders := map[string]string{
		tidutils.TransactionIDHeader: tID,
		originSystemIdHeader:         originSystemId,
	}

	writeLog.Info("write native content to content RW ...")
	err = h.contentRW.Write(ctx, uuid, &draftContent, draftHeaders)
	if err != nil {
		writeLog.WithError(err).Error("Error in writing draft content")
		writeMessage(w, fmt.Sprintf("Error in writing draft content: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func validateUUID(u string) error {
	_, err := uuid.FromString(u)
	return err
}

func validateOrigin(id string) (string, error) {
	var err error
	if _, found := allowedOriginSystemIdValues[id]; !found {
		err = errors.New(fmt.Sprintf("unsupported or missing value for X-Origin-System-Id: %v", id))
	}

	return id, err
}

func errorMessageForRead(status int) string {
	switch status {
	case http.StatusNotFound:
		return "Draft not found"

	case http.StatusUnprocessableEntity:
		return "Draft cannot be mapped into UPP format"
	}

	return "Error reading draft content"
}

func writeMessage(w http.ResponseWriter, errMsg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	jsonMsg := fmt.Sprintf(`{"message": "%v"}`, errMsg)
	w.Write([]byte(jsonMsg))
}
