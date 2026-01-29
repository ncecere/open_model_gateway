package adminprovider

import (
	"context"
	"sort"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
)

// Service exposes provider registry metadata for admin consumers.
type Service struct{}

// Definition describes a registered provider adapter.
type Definition struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Capabilities []string    `json:"capabilities"`
	Descriptor   *Descriptor `json:"descriptor,omitempty"`
}

// Descriptor documents configuration and entry-level inputs for a provider.
type Descriptor struct {
	Summary      string   `json:"summary"`
	Auth         []string `json:"auth,omitempty"`
	ConfigInputs []Input  `json:"config_inputs,omitempty"`
	EntryFields  []Input  `json:"entry_fields,omitempty"`
	HealthNotes  string   `json:"health_notes,omitempty"`
}

// Input captures a single provider configuration field.
type Input struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Source      string `json:"source,omitempty"`
}

// NewService returns a provider metadata service.
func NewService(_ *config.Config) *Service {
	return &Service{}
}

// List returns all registered providers sorted by name.
func (s *Service) List(ctx context.Context) ([]Definition, error) {
	_ = ctx
	defs := providers.DefaultDefinitions()
	out := make([]Definition, 0, len(defs))
	for _, def := range defs {
		caps := append([]string(nil), def.Capabilities...)
		sort.Strings(caps)
		d := Definition{
			Name:         def.Name,
			Description:  def.Description,
			Capabilities: caps,
		}
		// Expose descriptor metadata when available.
		desc := def.Descriptor
		if desc.Summary != "" || len(desc.ConfigInputs) > 0 || len(desc.EntryFields) > 0 {
			apiDesc := &Descriptor{
				Summary:     desc.Summary,
				Auth:        desc.Auth,
				HealthNotes: desc.HealthNotes,
			}
			for _, inp := range desc.ConfigInputs {
				apiDesc.ConfigInputs = append(apiDesc.ConfigInputs, Input{
					Name:        inp.Name,
					Description: inp.Description,
					Required:    inp.Required,
					Secret:      inp.Secret,
					Source:      inp.Source,
				})
			}
			for _, inp := range desc.EntryFields {
				apiDesc.EntryFields = append(apiDesc.EntryFields, Input{
					Name:        inp.Name,
					Description: inp.Description,
					Required:    inp.Required,
					Secret:      inp.Secret,
					Source:      inp.Source,
				})
			}
			d.Descriptor = apiDesc
		}
		out = append(out, d)
	}
	return out, nil
}
