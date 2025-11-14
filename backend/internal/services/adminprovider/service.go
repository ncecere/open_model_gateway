package adminprovider

import (
	"context"
	"sort"

	"github.com/ncecere/open_model_gateway/backend/internal/providers"
)

// Service exposes provider registry metadata for admin consumers.
type Service struct{}

// Definition describes a registered provider adapter.
type Definition struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

// NewService returns a provider metadata service.
func NewService() *Service { return &Service{} }

// List returns all registered providers sorted by name.
func (s *Service) List(ctx context.Context) ([]Definition, error) {
	defs := providers.DefaultDefinitions()
	out := make([]Definition, 0, len(defs))
	for _, def := range defs {
		caps := append([]string(nil), def.Capabilities...)
		sort.Strings(caps)
		out = append(out, Definition{
			Name:         def.Name,
			Description:  def.Description,
			Capabilities: caps,
		})
	}
	return out, nil
}
