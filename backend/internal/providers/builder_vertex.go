package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

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
		Builder:      buildVertexRoute,
	})
}

func buildVertexRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)

	md := cloneMetadata(entry.Metadata)
	override := entry.ProviderOverrides.Vertex

	projectID := pickFirst(
		func() string {
			if override != nil {
				return override.ProjectID
			}
			return ""
		}(),
		md["gcp_project_id"],
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
		Health:   adapter.HealthCheck,
	}

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
