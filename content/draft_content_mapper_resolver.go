package content

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// DraftContentValidatorResolver manages the validators available for a given originId/content-type pair.
type DraftContentValidatorResolver interface {
	// ValidatorForContentType Resolves and returns a DraftContentValidator implementation if present.
	ValidatorForContentType(contentType string) (DraftContentValidator, error)
}

// NewDraftContentValidatorResolver returns a DraftContentValidatorResolver implementation
func NewDraftContentValidatorResolver(contentTypeToValidator map[string]DraftContentValidator) DraftContentValidatorResolver {
	return &draftContentValidatorResolver{contentTypeToValidator}
}

type draftContentValidatorResolver struct {
	contentTypeToValidator map[string]DraftContentValidator
}

// ValidatorForContentType implementation checks the content-type validation for a validator resolution.
func (resolver *draftContentValidatorResolver) ValidatorForContentType(contentType string) (DraftContentValidator, error) {

	contentType = stripMediaTypeParameters(contentType)
	validator, found := resolver.contentTypeToValidator[contentType]

	if !found {
		logrus.Infof("contentTypeMap: %v", resolver.contentTypeToValidator)
		return nil, fmt.Errorf("no validator configured for contentType: %s", contentType)
	}

	return validator, nil
}
