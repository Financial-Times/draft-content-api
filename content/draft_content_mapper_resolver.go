package content

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
)

// DraftContentMapperResolver manages the mappers available for a given originId/content-type pair.
type DraftContentMapperResolver interface {
	// Resolves and returns a DraftContentMapper implementation if present.
	MapperForOriginIdAndContentType(contentType string) (DraftContentMapper, error)
}

// NewDraftContentMapperResolver returns a DraftContentMapperResolver implementation
func NewDraftContentMapperResolver(contentTypeToMapper map[string]DraftContentMapper) DraftContentMapperResolver {
	return &draftContentMapperResolver{contentTypeToMapper}
}

type draftContentMapperResolver struct {
	contentTypeToMapper map[string]DraftContentMapper
}

// MapperForOriginIdAndContentType implementation checks the content-type mapping for a mapper resolution.
func (resolver *draftContentMapperResolver) MapperForOriginIdAndContentType(contentType string) (DraftContentMapper, error) {

	// note: i disagree with this approach it's no longer valid; it makes sense to get the mapper having the contentType from AuroraDB but it does not make any
	// trying with originID in case it fails with ContentType; The reason, originID identifies who creates content,
	// but not what content is being created, cct and spark originID are the same entity and can create different content e.g :  articles; CPH. This approach is prone to fail !
	// originID it used to be injective, not any more.
	contentType = stripMediaTypeParameters(contentType)
	mapper, found := resolver.contentTypeToMapper[contentType]

	if !found {
		logrus.Infof("contentTypeMap: %v", resolver.contentTypeToMapper)
		return nil, errors.New(fmt.Sprintf("no mappers configured for contentType: %s", contentType))
	}

	return mapper, nil
}
