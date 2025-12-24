package openapi

import _ "embed"

// SpecYAML embeds the canonical OpenAPI specification for the router.
//
//go:embed openapi.yaml
var SpecYAML []byte
