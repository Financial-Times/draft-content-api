package content

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

// DraftContentMapperResolver manages the mappers available for a given originId/content-type pair.
type DraftContentMapperResolver interface {

	// Resolves and returns a DraftContentMapper implementation if present.
	MapperForOriginIdAndContentType(originId string, contentType string) (DraftContentMapper, error)
}

// NewDraftContentMapperResolver returns a DraftContentMapperResolver implementation
func NewDraftContentMapperResolver(originIdToMapper map[string]DraftContentMapper, contentTypeToMapper map[string]DraftContentMapper) DraftContentMapperResolver {
	return &draftContentMapperResolver{originIdToMapper, contentTypeToMapper}
}

type draftContentMapperResolver struct {
	originIdToMapper    map[string]DraftContentMapper
	contentTypeToMapper map[string]DraftContentMapper
}

// MapperForOriginIdAndContentType implementation checks the content-type mapping for a mapper resolution.
// If no mapping, it fallback to originId mapping lookup. Lookup miss always generates an error instead of returning nil values for a mapper
func (resolver *draftContentMapperResolver) MapperForOriginIdAndContentType(originId string, contentType string) (DraftContentMapper, error) {

	contentType = stripMediaTypeParameters(contentType)
	mapper, found := resolver.contentTypeToMapper[contentType]

	if found {
		return mapper, nil
	}

	mapper, found = resolver.originIdToMapper[originId]

	if !found {
		logrus.Infof("originIdMap: %v, contentTypeMap: %v", resolver.originIdToMapper, resolver.contentTypeToMapper)
		return nil, errors.New(fmt.Sprintf("no mappers configured for contentType: %s and originId: %s", contentType, originId))
	}

	return mapper, nil
}
