package content

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDraftContentMapperResolver_MapperForOriginIdAndContentType(t *testing.T) {

	mam := NewDraftContentMapperService("methode-endpoint", http.DefaultClient)
	ucv := NewDraftContentMapperService("spark-validator-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(happyResolverConfig(mam, ucv))

	methodeMapper, err := resolver.MapperForOriginIdAndContentType("methode-web-pub", "application/andromeda; charset=klingon")

	assert.NoError(t, err, "Methode mapper does not rely on content-type but on originId")
	assert.NotNil(t, methodeMapper, "Not received an error, mapper can't be nil")

	sparkMapper, err := resolver.MapperForOriginIdAndContentType("cct", "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err, "Spark mapper relies on content-type and originId. Both are present")
	assert.NotNil(t, sparkMapper, "Not received an error, mapper can't be nil")
}

func TestDraftContentMapperResolver_MissingMethodeMapping(t *testing.T) {

	ucv := NewDraftContentMapperService("spark-validator-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(cctOnlyResolverConfig(ucv))

	mapper, err := resolver.MapperForOriginIdAndContentType("methode-web-pub", "application/json")

	assert.Error(t, err)
	assert.Nil(t, mapper)

	sparkMapper, err := resolver.MapperForOriginIdAndContentType("cct", "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err, "Spark mapper relies on content-type and originId. Both are present")
	assert.NotNil(t, sparkMapper, "Not received an error, mapper can't be nil")

}
func TestDraftContentMapperResolver_MissingSparkMapping(t *testing.T) {

	mam := NewDraftContentMapperService("methode-endpoint", http.DefaultClient)
	resolver := NewDraftContentMapperResolver(methodeOnlyResolverConfig(mam, "methode-web-pub"))

	mapper, err := resolver.MapperForOriginIdAndContentType("cct", "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, mapper)
}

func happyResolverConfig(mam DraftContentMapper, ucv DraftContentMapper) map[string]map[string]DraftContentMapper {
	return map[string]map[string]DraftContentMapper{
		"methode-web-pub": {AnyType: mam},
		"cct":             {"application/vnd.ft-upp-article+json": ucv},
	}

}

func cctOnlyResolverConfig(ucv DraftContentMapper) map[string]map[string]DraftContentMapper {
	return map[string]map[string]DraftContentMapper{
		"cct": {"application/vnd.ft-upp-article+json": ucv},
	}

}

func methodeOnlyResolverConfig(mam DraftContentMapper, originId string) map[string]map[string]DraftContentMapper {
	return map[string]map[string]DraftContentMapper{
		originId: {AnyType: mam},
	}
}
