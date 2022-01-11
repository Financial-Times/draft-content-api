package content

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDraftContentMapperResolver_MapperForContentType(t *testing.T) {
	ucv := NewSparkDraftContentMapperService("upp-article-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(happyResolverConfig(ucv))

	uppContentValidator, err := resolver.MapperForContentType("application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err, "UPP Validator relies on content-type and originId. Both are present")
	assert.Equal(t, ucv, uppContentValidator, "Should return the same instance impl of DraftContentMapper")
}

func TestDraftContentMapperResolver_MissingSparkMapping(t *testing.T) {
	resolver := NewDraftContentMapperResolver(map[string]DraftContentMapper{})

	mapper, err := resolver.MapperForContentType("application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, mapper)
}

func happyResolverConfig(ucv DraftContentMapper) (contentTypeToMapper map[string]DraftContentMapper) {
	return map[string]DraftContentMapper{
		contentTypeArticle: ucv,
	}
}

func cctOnlyResolverConfig(ucv DraftContentMapper) (contentTypeToMapper map[string]DraftContentMapper) {
	return map[string]DraftContentMapper{
		contentTypeArticle: ucv,
	}
}
