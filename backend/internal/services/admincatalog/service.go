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

// UpsertResult wraps the persisted entry with any normalization warnings.
type UpsertResult struct {
	Entry    db.ModelCatalog            `json:"entry"`
	Warnings []catalog.NormalizeWarning `json:"warnings,omitempty"`
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

// ValidateResult wraps a normalized entry preview without persisting.
type ValidateResult struct {
	Entry    config.ModelCatalogEntry   `json:"entry"`
	Warnings []catalog.NormalizeWarning `json:"warnings,omitempty"`
	Errors   []string                   `json:"errors,omitempty"`
}

// Validate runs the normalizer on a payload and returns the preview without saving.
func (s *Service) Validate(payload ModelPayload) (ValidateResult, error) {
	var errs []string
	if strings.TrimSpace(payload.Alias) == "" {
		errs = append(errs, "alias is required")
	}
	if strings.TrimSpace(payload.Provider) == "" {
		errs = append(errs, "provider is required")
	}
	if strings.TrimSpace(payload.ProviderModel) == "" {
		errs = append(errs, "provider_model is required")
	}

	var enabled *bool
	t := payload.Enabled
	enabled = &t

	cfgEntry := config.ModelCatalogEntry{
		Alias:             payload.Alias,
		Provider:          payload.Provider,
		ProviderModel:     payload.ProviderModel,
		ModelType:         payload.ModelType,
		ContextWindow:     payload.ContextWindow,
		MaxOutputTokens:   payload.MaxOutputTokens,
		Modalities:        payload.Modalities,
		SupportsTools:     payload.SupportsTools,
		PriceInput:        payload.PriceInput,
		PriceOutput:       payload.PriceOutput,
		Currency:          payload.Currency,
		Deployment:        payload.Deployment,
		Endpoint:          payload.Endpoint,
		APIKey:            payload.APIKey,
		APIVersion:        payload.APIVersion,
		Region:            payload.Region,
		Weight:            int(payload.Weight),
		Enabled:           enabled,
		Metadata:          payload.Metadata,
		PricingTiers:      payload.PricingTiers,
		ProviderOverrides: payload.ProviderOverrides,
	}

	result := catalog.NormalizeEntry(cfgEntry)
	return ValidateResult{
		Entry:    result.Entry,
		Warnings: result.Warnings,
		Errors:   errs,
	}, nil
}

// Upsert validates and saves a catalog entry, reloading the router afterwards.
func (s *Service) Upsert(ctx context.Context, payload ModelPayload) (db.ModelCatalog, error) {
	result, err := s.UpsertWithWarnings(ctx, payload)
	if err != nil {
		return db.ModelCatalog{}, err
	}
	return result.Entry, nil
}

// UpsertWithWarnings validates, normalizes, and saves a catalog entry, returning
// the persisted entry along with any normalization warnings. Only alias, provider,
// and provider_model are strictly required; all other fields have smart defaults.
func (s *Service) UpsertWithWarnings(ctx context.Context, payload ModelPayload) (UpsertResult, error) {
	if s == nil || s.repo == nil {
		return UpsertResult{}, ErrServiceUnavailable
	}

	// Minimal required-field checks before normalization.
	if strings.TrimSpace(payload.Alias) == "" {
		return UpsertResult{}, ErrAliasRequired
	}
	if strings.TrimSpace(payload.Provider) == "" {
		return UpsertResult{}, ErrProviderRequired
	}
	if strings.TrimSpace(payload.ProviderModel) == "" {
		return UpsertResult{}, ErrModelRequired
	}

	// Convert the payload into a config entry and run through the shared normalizer.
	var enabled *bool
	t := payload.Enabled
	enabled = &t

	cfgEntry := config.ModelCatalogEntry{
		Alias:             payload.Alias,
		Provider:          payload.Provider,
		ProviderModel:     payload.ProviderModel,
		ModelType:         payload.ModelType,
		ContextWindow:     payload.ContextWindow,
		MaxOutputTokens:   payload.MaxOutputTokens,
		Modalities:        payload.Modalities,
		SupportsTools:     payload.SupportsTools,
		PriceInput:        payload.PriceInput,
		PriceOutput:       payload.PriceOutput,
		Currency:          payload.Currency,
		Deployment:        payload.Deployment,
		Endpoint:          payload.Endpoint,
		APIKey:            payload.APIKey,
		APIVersion:        payload.APIVersion,
		Region:            payload.Region,
		Weight:            int(payload.Weight),
		Enabled:           enabled,
		Metadata:          payload.Metadata,
		PricingTiers:      payload.PricingTiers,
		ProviderOverrides: payload.ProviderOverrides,
	}

	result := catalog.NormalizeEntry(cfgEntry)
	e := result.Entry

	// Marshal JSON fields.
	modalitiesJSON, err := json.Marshal(e.Modalities)
	if err != nil {
		return UpsertResult{}, err
	}
	metadataJSON, err := json.Marshal(e.Metadata)
	if err != nil {
		return UpsertResult{}, err
	}
	providerConfigJSON, err := json.Marshal(e.ProviderOverrides)
	if err != nil {
		return UpsertResult{}, err
	}
	pricingJSON, err := json.Marshal(e.PricingTiers)
	if err != nil {
		return UpsertResult{}, err
	}

	params := db.UpsertModelCatalogEntryParams{
		Alias:              e.Alias,
		Provider:           e.Provider,
		ProviderModel:      e.ProviderModel,
		ModelType:          e.ModelType,
		ContextWindow:      e.ContextWindow,
		MaxOutputTokens:    e.MaxOutputTokens,
		ModalitiesJson:     modalitiesJSON,
		SupportsTools:      e.SupportsTools,
		PricingTiersJson:   pricingJSON,
		PriceInput:         decimal.NewFromFloat(e.PriceInput),
		PriceOutput:        decimal.NewFromFloat(e.PriceOutput),
		Currency:           e.Currency,
		Enabled:            e.IsEnabled(),
		TenantAssignable:   payload.TenantAssignable,
		Deployment:         e.Deployment,
		Endpoint:           e.Endpoint,
		ApiKey:             e.APIKey,
		ApiVersion:         e.APIVersion,
		Region:             e.Region,
		MetadataJson:       metadataJSON,
		Weight:             int32(e.Weight),
		ProviderConfigJson: providerConfigJSON,
		ManagedBy:          "ui",
	}

	entry, err := s.repo.UpsertModelCatalogEntry(ctx, params)
	if err != nil {
		return UpsertResult{}, err
	}
	if s.reload != nil {
		if err := s.reload(ctx); err != nil {
			return UpsertResult{}, err
		}
	}
	entry.ModelType = e.ModelType
	return UpsertResult{Entry: entry, Warnings: result.Warnings}, nil
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
