package content

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

// DraftContentMapperResolver manages the mappers available for a given originId/content-type pair.
type DraftContentMapperResolver interface {
	// Resolves and returns a DraftContentMapper implementation if present.
	MapperForContentType(contentType string) (DraftContentMapper, error)
}

// NewDraftContentMapperResolver returns a DraftContentMapperResolver implementation
func NewDraftContentMapperResolver(contentTypeToMapper map[string]DraftContentMapper) DraftContentMapperResolver {
	return &draftContentMapperResolver{contentTypeToMapper}
}

type draftContentMapperResolver struct {
	contentTypeToMapper map[string]DraftContentMapper
}

// MapperForContentType implementation checks the content-type mapping for a mapper resolution.
func (resolver *draftContentMapperResolver) MapperForContentType(contentType string) (DraftContentMapper, error) {

	contentType = stripMediaTypeParameters(contentType)
	mapper, found := resolver.contentTypeToMapper[contentType]

	if !found {
		logrus.Infof("contentTypeMap: %v", resolver.contentTypeToMapper)
		return nil, fmt.Errorf("no mappers configured for contentType: %s", contentType)
	}

	return mapper, nil
}
