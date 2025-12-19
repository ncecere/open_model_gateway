package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/adapters/vertex"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func pickFirst(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func init() {
	RegisterDefinition(Definition{
		Name:         "vertex",
		Description:  "Google Vertex AI (Gemini chat + embeddings + Imagen)",
		Capabilities: []string{"chat", "chat_stream", "embeddings", "images"},
		Descriptor: Descriptor{
			Summary: "Google Vertex AI via service account JSON (Gemini text+image models).",
			Auth:    []string{"gcp_service_account"},
			ConfigInputs: []Input{
				{Name: "providers.vertex.gcp_project_id", Description: "Default GCP project", Required: true, Source: "config.providers.vertex.gcp_project_id"},
				{Name: "providers.vertex.gcp_credentials_json", Description: "Base64 or JSON service account", Required: true, Secret: true, Source: "config.providers.vertex.gcp_credentials_json"},
				{Name: "providers.vertex.vertex_location", Description: "Default Vertex location", Source: "config.providers.vertex.vertex_location"},
			},
			EntryFields: []Input{
				{Name: "region", Description: "Vertex location", Source: "catalog.region"},
				{Name: "gcp_credentials_json", Description: "Override credentials", Secret: true, Source: "catalog.metadata.gcp_credentials_json"},
				{Name: "gcp_credentials_format", Description: "`json` or `base64`", Source: "catalog.metadata.gcp_credentials_format"},
			},
			RetryPolicy: RetryDescriptor{
				Description:   "Executor retries Vertex API calls twice; adapters unwrap Google errors with retry hints.",
				DefaultPolicy: "exponential 250ms backoff, max 2 attempts",
			},
			HealthNotes: "Lists available Vertex models using the configured credentials/location.",
		},
		Builder: buildVertexRoute,
	})
}

func buildVertexRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)

	md := cloneMetadata(entry.Metadata)
	override := entry.ProviderOverrides.Vertex
	vertexCfg := cfg.Providers.Vertex

	projectID := pickFirst(
		func() string {
			if override != nil {
				return override.ProjectID
			}
			return ""
		}(),
		md["gcp_project_id"],
		vertexCfg.ProjectID,
		cfg.Providers.GCPProjectID,
	)
	if projectID == "" {
		return Route{}, fmt.Errorf("vertex provider requires gcp_project_id")
	}

	location := pickFirst(
		func() string {
			if override != nil {
				return override.Location
			}
			return ""
		}(),
		entry.Region,
		md["vertex_location"],
		vertexCfg.Location,
	)
	if location == "" {
		location = "us-central1"
	}

	credSource := pickFirst(
		func() string {
			if override != nil {
				return override.CredentialsJSON
			}
			return ""
		}(),
		md["gcp_credentials_json"],
		vertexCfg.CredentialsJSON,
		cfg.Providers.GCPJSONCredentials,
	)
	if credSource == "" {
		return Route{}, fmt.Errorf("vertex provider requires gcp credentials json")
	}
	credSource = strings.TrimSpace(credSource)

	format := pickFirst(
		func() string {
			if override != nil {
				return override.CredentialsFormat
			}
			return ""
		}(),
		md["gcp_credentials_format"],
		vertexCfg.CredentialsFormat,
	)
	credBytes := []byte(credSource)
	switch strings.ToLower(format) {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(credSource)
		if err != nil {
			return Route{}, fmt.Errorf("vertex credentials base64 decode: %w", err)
		}
		if !json.Valid(decoded) {
			return Route{}, fmt.Errorf("vertex credentials base64 decode produced invalid JSON")
		}
		credBytes = decoded
	case "json", "":
		if !json.Valid(credBytes) {
			if decoded, err := base64.StdEncoding.DecodeString(credSource); err == nil && json.Valid(decoded) {
				credBytes = decoded
				format = "base64"
			} else {
				return Route{}, fmt.Errorf("vertex credentials json invalid or truncated")
			}
		}
	default:
		return Route{}, fmt.Errorf("vertex credentials format %q not supported", format)
	}

	opts := vertex.Options{
		ProjectID: projectID,
		Location:  location,
		Publisher: func() string {
			if override != nil && strings.TrimSpace(override.Publisher) != "" {
				return strings.TrimSpace(override.Publisher)
			}
			if strings.TrimSpace(vertexCfg.Publisher) != "" {
				return strings.TrimSpace(vertexCfg.Publisher)
			}
			return md["vertex_publisher"]
		}(),
		Model:           entry.ProviderModel,
		Endpoint:        entry.Endpoint,
		CredentialsJSON: credBytes,
		Metadata:        md,
	}

	md["gcp_project_id"] = projectID
	md["vertex_location"] = location

	adapter, err := vertex.New(ctx, opts)
	if err != nil {
		return Route{}, err
	}

	weight := entry.Weight
	if weight == 0 {
		weight = 100
	}

	route := Route{
		Alias:    entry.Alias,
		Provider: entry.Provider,
		Model:    entry.ProviderModel,
		Weight:   weight,
		Metadata: md,
		Health:   WrapHealth(adapter.HealthCheck),
	}
	route.Retry = mergeRetry(RetryConfig{MaxAttempts: 2, InitialBackoff: 300 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
	route.Tokenizer = selectTokenizer("vertex", entry.ProviderOverrides.Tokenizer, route.Metadata)

	if supportsModality(entry.Modalities, "text") {
		route.Chat = adapter
		route.ChatStream = adapter
	}
	if supportsEmbedding(entry.Modalities) {
		route.Embedding = adapter
	}
	if supportsModality(entry.Modalities, "image") {
		route.Image = adapter
	}

	if route.Chat == nil && route.Embedding == nil && route.Image == nil {
		return Route{}, fmt.Errorf("vertex route %s has no supported modalities", entry.Alias)
	}

	return route, nil
}
