package content

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDraftContentMapperResolver_MapperForOriginIdAndContentType(t *testing.T) {

	mam := NewDraftContentMapperService("methode-endpoint", http.DefaultClient)
	ucv := NewDraftContentMapperService("upp-content-validator-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(happyResolverConfig(mam, ucv))

	methodeMapper, err := resolver.MapperForOriginIdAndContentType("methode-web-pub", "application/andromeda; charset=klingon")

	assert.NoError(t, err, "Fallback to originId lookup should've handled the content-type lookup miss")
	assert.Equal(t, mam, methodeMapper, "Should return the same instance impl of DraftContentMapper")

	uppContentValidator, err := resolver.MapperForOriginIdAndContentType("cct", "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err, "UPP Validator relies on content-type and originId. Both are present")
	assert.Equal(t, ucv, uppContentValidator, "Should return the same instance impl of DraftContentMapper")
}

func TestDraftContentMapperResolver_MissingMethodeMapping(t *testing.T) {

	ucv := NewDraftContentMapperService("upp-content-validator-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(cctOnlyResolverConfig(ucv))

	mapper, err := resolver.MapperForOriginIdAndContentType("methode-web-pub", "application/json")

	assert.Error(t, err)
	assert.Nil(t, mapper)

	uppContentValidator, err := resolver.MapperForOriginIdAndContentType("cct", "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err, "Fallback to originId lookup should've handled the content-type lookup miss")
	assert.Equal(t, ucv, uppContentValidator, "Should return the same instance impl of DraftContentMapper")

}
func TestDraftContentMapperResolver_MissingSparkMapping(t *testing.T) {

	mam := NewDraftContentMapperService("methode-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(methodeOnlyResolverConfig(mam, "methode-web-pub"))

	mapper, err := resolver.MapperForOriginIdAndContentType("cct", "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, mapper)
}

func happyResolverConfig(mam DraftContentMapper, ucv DraftContentMapper) (originIdToMapper map[string]DraftContentMapper, contentTypeToMapper map[string]DraftContentMapper) {
	return map[string]DraftContentMapper{
			"methode-web-pub": mam,
		}, map[string]DraftContentMapper{
			"application/vnd.ft-upp-article+json": ucv,
		}
}

func cctOnlyResolverConfig(ucv DraftContentMapper) (originIdToMapper map[string]DraftContentMapper, contentTypeToMapper map[string]DraftContentMapper) {
	return map[string]DraftContentMapper{}, map[string]DraftContentMapper{
		"application/vnd.ft-upp-article+json": ucv,
	}
}

func methodeOnlyResolverConfig(mam DraftContentMapper, originId string) (originIdToMapper map[string]DraftContentMapper, contentTypeToMapper map[string]DraftContentMapper) {
	return map[string]DraftContentMapper{
		originId: mam,
	}, map[string]DraftContentMapper{}
}
