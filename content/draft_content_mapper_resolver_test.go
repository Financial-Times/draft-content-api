package content

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDraftContentValidatorResolver_ValidatorForContentType(t *testing.T) {
	ucv := NewSparkDraftContentValidatorService("upp-article-endpoint", http.DefaultClient)
	resolver := NewDraftContentValidatorResolver(cctOnlyResolverConfig(ucv))

	uppContentValidator, err := resolver.ValidatorForContentType("application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err, "UPP Validator relies on content-type and originId. Both are present")
	assert.Equal(t, ucv, uppContentValidator, "Should return the same instance impl of DraftContentValidator")
}

func TestDraftContentValidatorResolver_MissingSparkValidation(t *testing.T) {
	resolver := NewDraftContentValidatorResolver(map[string]DraftContentValidator{})

	validator, err := resolver.ValidatorForContentType("application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, validator)
}

func cctOnlyResolverConfig(ucv DraftContentValidator) (contentTypeToValidator map[string]DraftContentValidator) {
	return map[string]DraftContentValidator{
		contentTypeArticle: ucv,
	}
}
