package content

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDraftContentMapperResolver_MapperForContentType(t *testing.T) {

	mam := NewDraftContentMapperService("methode-endpoint", http.DefaultClient)
	ucv := NewDraftContentMapperService("upp-article-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(happyResolverConfig(mam, ucv))

	methodeMapper, err := resolver.MapperForContentType(contentType)

	assert.NoError(t, err, "Fallback to originId lookup should've handled the content-type lookup miss")
	assert.Equal(t, mam, methodeMapper, "Should return the same instance impl of DraftContentMapper")

	uppContentValidator, err := resolver.MapperForContentType("application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err, "UPP Validator relies on content-type and originId. Both are present")
	assert.Equal(t, ucv, uppContentValidator, "Should return the same instance impl of DraftContentMapper")
}

func TestDraftContentMapperResolver_MissingMethodeMapping(t *testing.T) {

	ucv := NewDraftContentMapperService("upp-article-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(cctOnlyResolverConfig(ucv))

	mapper, err := resolver.MapperForContentType(contentType)

	assert.Error(t, err)
	assert.Nil(t, mapper)

	uppContentValidator, err := resolver.MapperForContentType("application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err, "Fallback to originId lookup should've handled the content-type lookup miss")
	assert.Equal(t, ucv, uppContentValidator, "Should return the same instance impl of DraftContentMapper")

}

func TestDraftContentMapperResolver_MissingSparkMapping(t *testing.T) {

	mam := NewDraftContentMapperService("methode-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(methodeOnlyResolverConfig(mam))

	mapper, err := resolver.MapperForContentType("application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, mapper)
}

func happyResolverConfig(mam DraftContentMapper, ucv DraftContentMapper) (contentTypeToMapper map[string]DraftContentMapper) {
	return map[string]DraftContentMapper{
		contentType:        mam,
		contentTypeArticle: ucv,
	}
}

func cctOnlyResolverConfig(ucv DraftContentMapper) (contentTypeToMapper map[string]DraftContentMapper) {
	return map[string]DraftContentMapper{
		contentTypeArticle: ucv,
	}
}

func methodeOnlyResolverConfig(mam DraftContentMapper) (contentTypeToMapper map[string]DraftContentMapper) {
	return map[string]DraftContentMapper{
		contentType: mam,
	}
}
