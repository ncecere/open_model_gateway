package admincatalog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

var (
	ErrServiceUnavailable = errors.New("admin catalog service not initialized")
	ErrAliasRequired      = errors.New("alias is required")
	ErrProviderRequired   = errors.New("provider is required")
	ErrModelRequired      = errors.New("provider_model is required")
	ErrDeploymentRequired = errors.New("deployment is required")
)

// ReloadFunc triggers a router reload after catalog changes.
type ReloadFunc func(ctx context.Context) error

// Service wraps admin model catalog operations.
type Service struct {
	repo   Repository
	reload ReloadFunc
}

// NewService constructs a catalog service.
// Deprecated: Use NewServiceWithRepository for new code.
func NewService(queries *db.Queries, reload ReloadFunc) *Service {
	return NewServiceWithRepository(NewQueriesRepository(queries), reload)
}

// NewServiceWithRepository constructs a catalog service with a Repository interface.
func NewServiceWithRepository(repo Repository, reload ReloadFunc) *Service {
	return &Service{repo: repo, reload: reload}
}

// ModelPayload represents the upsert request body.
type ModelPayload struct {
	Alias            string              `json:"alias"`
	Provider         string              `json:"provider"`
	ProviderModel    string              `json:"provider_model"`
	ModelType        string              `json:"model_type"`
	ContextWindow    int32               `json:"context_window"`
	MaxOutputTokens  int32               `json:"max_output_tokens"`
	Modalities       []string            `json:"modalities"`
	SupportsTools    bool                `json:"supports_tools"`
	PriceInput       float64             `json:"price_input"`
	PriceOutput      float64             `json:"price_output"`
	Currency         string              `json:"currency"`
	Deployment       string              `json:"deployment"`
	Endpoint         string              `json:"endpoint"`
	APIKey           string              `json:"api_key"`
	APIVersion       string              `json:"api_version"`
	Region           string              `json:"region"`
	Weight           int32               `json:"weight"`
	Enabled          bool                `json:"enabled"`
	TenantAssignable bool                `json:"tenant_assignable"`
	Metadata         map[string]string   `json:"metadata"`
	PricingTiers     config.PricingTiers `json:"pricing_tiers"`
	config.ProviderOverrides
}

// List returns the model catalog entries.
func (s *Service) List(ctx context.Context) ([]db.ModelCatalog, error) {
	if s == nil || s.repo == nil {
		return nil, ErrServiceUnavailable
	}
	items, err := s.repo.ListModelCatalog(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Provider = catalog.NormalizeProviderSlug(items[i].Provider)
	}
	return items, nil
}

// Upsert validates and saves a catalog entry, reloading the router afterwards.
func (s *Service) Upsert(ctx context.Context, payload ModelPayload) (db.ModelCatalog, error) {
	if s == nil || s.repo == nil {
		return db.ModelCatalog{}, ErrServiceUnavailable
	}
	alias := strings.TrimSpace(payload.Alias)
	if alias == "" {
		return db.ModelCatalog{}, ErrAliasRequired
	}
	provider := catalog.NormalizeProviderSlug(payload.Provider)
	if provider == "" {
		return db.ModelCatalog{}, ErrProviderRequired
	}
	modelType := catalog.NormalizeModelType(payload.ModelType)
	if modelType == "" {
		modelType = "llm"
	}
	model := strings.TrimSpace(payload.ProviderModel)
	if model == "" {
		return db.ModelCatalog{}, ErrModelRequired
	}
	deployment := strings.TrimSpace(payload.Deployment)
	if payload.Weight == 0 {
		payload.Weight = 100
	}
	if payload.Metadata == nil {
		payload.Metadata = map[string]string{}
	}
	if payload.PricingTiers == nil {
		payload.PricingTiers = config.PricingTiers{}
	}

	switch provider {
	case "azure":
		if cfg := payload.ProviderOverrides.Azure; cfg != nil {
			if deployment == "" {
				deployment = strings.TrimSpace(cfg.Deployment)
			}
			if payload.Endpoint == "" {
				payload.Endpoint = strings.TrimSpace(cfg.Endpoint)
			}
			if payload.APIKey == "" {
				payload.APIKey = strings.TrimSpace(cfg.APIKey)
			}
			if payload.APIVersion == "" {
				payload.APIVersion = strings.TrimSpace(cfg.APIVersion)
			}
			if payload.Region == "" {
				payload.Region = strings.TrimSpace(cfg.Region)
			}
		}
	case "openai":
		if cfg := payload.ProviderOverrides.OpenAI; cfg != nil {
			if payload.Endpoint == "" {
				payload.Endpoint = strings.TrimSpace(cfg.BaseURL)
			}
			if payload.APIKey == "" {
				payload.APIKey = strings.TrimSpace(cfg.APIKey)
			}
		}
	case "openai-compatible":
		if cfg := payload.ProviderOverrides.OpenAICompatible; cfg != nil {
			if payload.Endpoint == "" {
				payload.Endpoint = strings.TrimSpace(cfg.BaseURL)
			}
			if payload.APIKey == "" {
				payload.APIKey = strings.TrimSpace(cfg.APIKey)
			}
		}
	case "openrouter":
		if cfg := payload.ProviderOverrides.OpenRouter; cfg != nil {
			if payload.Endpoint == "" {
				payload.Endpoint = strings.TrimSpace(cfg.BaseURL)
			}
			if payload.APIKey == "" {
				payload.APIKey = strings.TrimSpace(cfg.APIKey)
			}
		}
	case "vllm":
		if cfg := payload.ProviderOverrides.VLLM; cfg != nil {
			if payload.Endpoint == "" {
				payload.Endpoint = strings.TrimSpace(cfg.BaseURL)
			}
			if payload.APIKey == "" {
				payload.APIKey = strings.TrimSpace(cfg.APIKey)
			}
		}
	case "bedrock":
		if cfg := payload.ProviderOverrides.Bedrock; cfg != nil && payload.Region == "" {
			payload.Region = strings.TrimSpace(cfg.Region)
		}
	case "vertex":
		if cfg := payload.ProviderOverrides.Vertex; cfg != nil && payload.Region == "" {
			payload.Region = strings.TrimSpace(cfg.Location)
		}
	}

	if deployment == "" {
		return db.ModelCatalog{}, ErrDeploymentRequired
	}

	endpoint := strings.TrimSpace(payload.Endpoint)
	apiKey := strings.TrimSpace(payload.APIKey)
	apiVersion := strings.TrimSpace(payload.APIVersion)
	region := strings.TrimSpace(payload.Region)

	modalitiesJSON, err := json.Marshal(payload.Modalities)
	if err != nil {
		return db.ModelCatalog{}, err
	}
	metadataJSON, err := json.Marshal(payload.Metadata)
	if err != nil {
		return db.ModelCatalog{}, err
	}
	providerConfigJSON, err := json.Marshal(payload.ProviderOverrides)
	if err != nil {
		return db.ModelCatalog{}, err
	}
	pricingJSON, err := json.Marshal(payload.PricingTiers)
	if err != nil {
		return db.ModelCatalog{}, err
	}

	params := db.UpsertModelCatalogEntryParams{
		Alias:              alias,
		Provider:           provider,
		ProviderModel:      model,
		ModelType:          modelType,
		ContextWindow:      payload.ContextWindow,
		MaxOutputTokens:    payload.MaxOutputTokens,
		ModalitiesJson:     modalitiesJSON,
		SupportsTools:      payload.SupportsTools,
		PricingTiersJson:   pricingJSON,
		PriceInput:         decimal.NewFromFloat(payload.PriceInput),
		PriceOutput:        decimal.NewFromFloat(payload.PriceOutput),
		Currency:           strings.ToUpper(strings.TrimSpace(payload.Currency)),
		Enabled:            payload.Enabled,
		TenantAssignable:   payload.TenantAssignable,
		Deployment:         deployment,
		Endpoint:           endpoint,
		ApiKey:             apiKey,
		ApiVersion:         apiVersion,
		Region:             region,
		MetadataJson:       metadataJSON,
		Weight:             payload.Weight,
		ProviderConfigJson: providerConfigJSON,
	}
	if params.Currency == "" {
		params.Currency = "USD"
	}

	entry, err := s.repo.UpsertModelCatalogEntry(ctx, params)
	if err != nil {
		return db.ModelCatalog{}, err
	}
	if s.reload != nil {
		if err := s.reload(ctx); err != nil {
			return db.ModelCatalog{}, err
		}
	}
	entry.ModelType = modelType
	return entry, nil
}

// Remove deletes an entry and reloads the router.
func (s *Service) Remove(ctx context.Context, alias string) error {
	if s == nil || s.repo == nil {
		return ErrServiceUnavailable
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return ErrAliasRequired
	}
	if err := s.repo.DeleteModelCatalogEntry(ctx, alias); err != nil {
		return err
	}
	if s.reload != nil {
		return s.reload(ctx)
	}
	return nil
}
