package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/husobee/vestigo"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	typePrefix = "http://www.ft.com/ontology/content/"
	idPrefix   = "http://www.ft.com/thing/"
)

type Handler struct {
	uppContentAPI ContentAPI
	contentRW     DraftContentRW
}

func NewHandler(uppApi ContentAPI, draftContentRW DraftContentRW) *Handler {
	return &Handler{uppApi, draftContentRW}
}

func (h *Handler) ReadContent(w http.ResponseWriter, r *http.Request) {
	contentId := vestigo.Param(r, "uuid")
	tID := tidutils.GetTransactionIDFromRequest(r)
	ctx := tidutils.TransactionAwareContext(context.Background(), tID)

	content, err := h.contentRW.Read(ctx, contentId)

	if err == nil {
		defer content.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.Copy(w, content)
		return
	}

	if err == ErrDraftNotMappable {
		writeMessage(w, errorMessageForRead(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	if err == ErrDraftNotFound {
		h.readContentFromUPP(ctx, w, contentId)
		return
	}

	writeMessage(w, errorMessageForRead(http.StatusInternalServerError), http.StatusInternalServerError)

}

func (h *Handler) WriteNativeContent(w http.ResponseWriter, r *http.Request) {
	contentId := vestigo.Param(r, "uuid")
	tID := tidutils.GetTransactionIDFromRequest(r)
	ctx := tidutils.TransactionAwareContext(context.Background(), tID)

	writeLog := log.WithField(tidutils.TransactionIDKey, tID).WithField("uuid", contentId)

	if err := validateUUID(contentId); err != nil {
		writeLog.WithError(err).Error("Invalid content UUID")
		writeMessage(w, fmt.Sprintf("Invalid content UUID: %v", contentId), http.StatusBadRequest)
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
	err = h.contentRW.Write(ctx, contentId, &draftContent, draftHeaders)
	if err != nil {
		writeLog.WithError(err).Error("Error in writing draft content")
		writeMessage(w, fmt.Sprintf("Error in writing draft content: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) readContentFromUPP(ctx context.Context, w http.ResponseWriter, contentId string) {

	readContentUPPLog := log.WithField(tidutils.TransactionIDKey, ctx.Value(tidutils.TransactionIDKey)).WithField("uuid", contentId)
	readContentUPPLog.Warn("Draft not found in PAC, trying UPP")
	uppResp, err := h.uppContentAPI.Get(ctx, contentId)

	if err != nil {
		readContentUPPLog.WithError(err).Error("Error in calling Content API")
		writeMessage(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer uppResp.Body.Close()

	if uppResp.StatusCode == http.StatusGatewayTimeout {
		writeMessage(w, errorMessageForRead(uppResp.StatusCode), http.StatusInternalServerError)
		return
	}

	if uppResp.StatusCode != http.StatusOK {
		writeMessage(w, errorMessageForRead(uppResp.StatusCode), uppResp.StatusCode)
		return
	}

	bytes, err := ioutil.ReadAll(uppResp.Body)

	if err != nil {
		readContentUPPLog.WithError(err).Error("Failed reading UPP response")
		writeMessage(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var uppContent map[string]interface{}

	err = json.Unmarshal(bytes, &uppContent)

	if err != nil {
		readContentUPPLog.WithError(err).Error("Failed marshalling UPP response")
		writeMessage(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = transformUPPContent(uppContent)

	if err != nil {
		readContentUPPLog.WithError(err).Error("Failed transforming UPP response")
		writeMessage(w, err.Error(), http.StatusInternalServerError)
		return
	}

	content, err := json.Marshal(uppContent)

	if err != nil {
		readContentUPPLog.WithError(err).Error("Failed marshalling transformed UPP response")
		writeMessage(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
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

// Function modifies these fields/values;
//  - id -> uuid, removing the http prefix
//  - bodyXml -> body, keeping value intact
//  - type value, adding http prefix
//  - brands value, adding an object wrapper with id field having the same value
func transformUPPContent(content map[string]interface{}) error {

	// --- uuid
	if id, present := content["id"]; present {
		uniqueId, assertion := id.(string)

		if !assertion {
			return errors.New("invalid id value, was expecting string")
		}

		delete(content, "id")
		uniqueId = strings.Replace(uniqueId, idPrefix, "", 1)
		content["uuid"] = uniqueId
	}

	// --- body
	if _, present := content["body"]; present {
		content["body"] = content["bodyXml"]
		delete(content, "body")
	}

	// --- type
	if contentType, present := content["type"]; present {
		contentType, assertion := contentType.(string)

		if !assertion {
			return errors.New("invalid type value, was expecting string")
		}

		content["type"] = strings.Replace(contentType, typePrefix, "", 1)
	}

	// --- brands
	if brands, present := content["brands"]; present {

		var idBrandTuples []map[string]string

		brands, assertion := brands.([]interface{})

		if !assertion {
			return errors.New("invalid brands value, was expecting array")
		}

		for _, brand := range brands {
			brand, assertion := brand.(string)

			if !assertion {
				return errors.New("invalid brand entry, was expecting string")
			}

			idBrandTuples = append(idBrandTuples, map[string]string{"id": brand})

		}

		content["brands"] = idBrandTuples

	}

	return nil

}
