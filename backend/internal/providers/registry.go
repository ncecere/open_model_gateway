package providers

import (
	"sort"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// Definition captures the metadata required to register a provider builder.
type Definition struct {
	Name         string
	Description  string
	Capabilities []string
	Builder      Builder
	Descriptor   Descriptor
}

// Descriptor documents configuration inputs, authentication modes, retry policy,
// and other metadata for provider builders. This powers documentation, CLI help,
// and future validation tooling.
type Descriptor struct {
	Summary      string
	Auth         []string
	ConfigInputs []Input
	EntryFields  []Input
	RetryPolicy  RetryDescriptor
	HealthNotes  string
}

// Input captures a provider configuration field (global or per-entry).
type Input struct {
	Name        string
	Description string
	Required    bool
	Secret      bool
	Source      string
}

// RetryDescriptor documents default retry/backoff behaviour for a provider.
type RetryDescriptor struct {
	Description   string
	DefaultPolicy string
}

var defaultDefinitions = map[string]Definition{}

// RegisterDefaultBuilder is kept for backwards compatibility; prefer RegisterDefinition.
func RegisterDefaultBuilder(name string, builder Builder) {
	RegisterDefinition(Definition{Name: name, Builder: builder})
}

// RegisterDefinition stores a provider definition so factories can resolve builders by name.
func RegisterDefinition(def Definition) {
	if def.Builder == nil {
		panic("providers: definition builder required")
	}
	if def.Name == "" {
		panic("providers: definition name required")
	}
	if def.Description == "" {
		def.Description = def.Name
	}
	if def.Descriptor.Summary == "" {
		def.Descriptor.Summary = def.Description
	}
	if len(def.Capabilities) > 0 {
		caps := make([]string, len(def.Capabilities))
		copy(caps, def.Capabilities)
		sort.Strings(caps)
		def.Capabilities = caps
	}
	if defaultDefinitions == nil {
		defaultDefinitions = make(map[string]Definition)
	}
	defaultDefinitions[def.Name] = def
}

// DefaultDefinitions returns the registered provider definitions sorted by name (useful for docs/tests).
func DefaultDefinitions() []Definition {
	defs := make([]Definition, 0, len(defaultDefinitions))
	for _, def := range defaultDefinitions {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}

// DefaultDescriptors returns the descriptors attached to registered providers.
func DefaultDescriptors() []Descriptor {
	defs := DefaultDefinitions()
	descriptors := make([]Descriptor, len(defs))
	for i, def := range defs {
		descriptors[i] = def.Descriptor
	}
	return descriptors
}

func cloneDefaultBuilders() map[string]Builder {
	builders := make(map[string]Builder, len(defaultDefinitions))
	for name, def := range defaultDefinitions {
		builders[name] = def.Builder
	}
	return builders
}

// EnsureConfig ensures the config pointer is not nil when builders run.
func EnsureConfig(cfg *config.Config) *config.Config {
	if cfg == nil {
		panic("providers: config is required")
	}
	return cfg
}
