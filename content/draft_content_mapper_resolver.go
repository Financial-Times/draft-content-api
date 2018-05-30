package content

import (
	"errors"
	"fmt"
)

const (
	AnyType = "*/*"
)

// DraftContentMapperResolver manages the mappers available for a given originId/content-type pair.
type DraftContentMapperResolver interface {

	// Resolves and returns a DraftContentMapper implementation if present.
	// It also has '*/*' wildcard support for resolving to a more default/generic mapper for a
	// specific originId.
	MapperForOriginIdAndContentType(originId string, contentType string) (DraftContentMapper, error)
}

// NewDraftContentMapperResolver returns a DraftContentMapperResolver implementation
func NewDraftContentMapperResolver(config map[string]map[string]DraftContentMapper) DraftContentMapperResolver {
	return &draftContentMapperResolver{config: config}
}

type draftContentMapperResolver struct {
	config map[string]map[string]DraftContentMapper
}

func (resolver *draftContentMapperResolver) MapperForOriginIdAndContentType(originId string, contentType string) (DraftContentMapper, error) {

	contentMappers, found := resolver.config[originId]
	if !found {
		return nil, errors.New(fmt.Sprintf("no mappers configured for originId: %s", originId))
	}

	contentType = stripMediaTypeParameters(contentType)

	mapper, found := contentMappers[contentType]
	if !found {

		mapper, found = contentMappers[AnyType]

		if !found {
			return nil, errors.New(fmt.Sprintf("no mappers configured for contentType: %s", contentType))
		}
	}

	return mapper, nil

}
