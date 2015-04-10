package graph

import (
	"strings"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/utils"
)

func (s *TagStore) History(name string) ([]*types.ImageHistory, error) {
	foundImage, err := s.LookupImage(name)
	if err != nil {
		return nil, err
	}

	lookupMap := make(map[string][]string)
	for name, repository := range s.Repositories {
		for tag, id := range repository {
			// If the ID already has a reverse lookup, do not update it unless for "latest"
			if _, exists := lookupMap[id]; !exists {
				lookupMap[id] = []string{}
			}
			lookupMap[id] = append(lookupMap[id], utils.ImageReference(name, tag))
		}
	}

	history := []*types.ImageHistory{}

	err = foundImage.WalkHistory(func(img *image.Image) error {
		history = append(history, &types.ImageHistory{
			ID:        img.ID,
			Created:   img.Created.Unix(),
			CreatedBy: strings.Join(img.ContainerConfig.Cmd, " "),
			Tags:      lookupMap[img.ID],
			Size:      img.Size,
			Comment:   img.Comment,
		})
		return nil
	})

	return history, err
}
