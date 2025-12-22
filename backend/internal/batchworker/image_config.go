package batchworker

import (
	"context"
	"strings"

	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
)

func newImageGenerationOperationConfig(req models.ImageRequest) executor.ImageOperationConfig {
	baseReq := req
	size := strings.TrimSpace(req.Size)
	return executor.ImageOperationConfig{
		ImagePixels:     models.ImagePixelCount(size, req.N),
		PricingMetadata: buildImagePricingMetadata(imageOperationGeneration, req.Quality, size),
		Builder: func(callCtx context.Context, route providers.Route) (models.ImageResponse, error) {
			modelReq := baseReq
			modelReq.Model = route.ResolveDeployment()
			return route.Image.Generate(callCtx, modelReq)
		},
		OverrideCost: func(metadata map[string]string) *int64 {
			return parseImageOverrideCost(metadata, imageOperationGeneration)
		},
	}
}

func newImageEditOperationConfig(req models.ImageEditRequest) executor.ImageOperationConfig {
	baseReq := req
	size := strings.TrimSpace(req.Size)
	if size == "" && len(req.Images) > 0 {
		if derived, ok := models.ImageSizeFromInput(req.Images[0]); ok {
			size = derived
		}
	}
	return executor.ImageOperationConfig{
		ImagePixels:     models.ImagePixelCount(size, req.N),
		PricingMetadata: buildImagePricingMetadata(imageOperationEdit, req.Quality, size),
		Builder: func(callCtx context.Context, route providers.Route) (models.ImageResponse, error) {
			modelReq := baseReq
			modelReq.Model = route.ResolveDeployment()
			modelReq.Images = cloneImageInputs(baseReq.Images)
			if baseReq.Mask != nil {
				maskCopy := *baseReq.Mask
				modelReq.Mask = &maskCopy
			}
			return route.Image.Edit(callCtx, modelReq)
		},
		OverrideCost: func(metadata map[string]string) *int64 {
			return parseImageOverrideCost(metadata, imageOperationEdit)
		},
	}
}

func newImageVariationOperationConfig(req models.ImageVariationRequest) executor.ImageOperationConfig {
	baseReq := req
	size := strings.TrimSpace(req.Size)
	if size == "" {
		if derived, ok := models.ImageSizeFromInput(req.Image); ok {
			size = derived
		}
	}
	return executor.ImageOperationConfig{
		ImagePixels:     models.ImagePixelCount(size, req.N),
		PricingMetadata: buildImagePricingMetadata(imageOperationVariation, req.Quality, size),
		Builder: func(callCtx context.Context, route providers.Route) (models.ImageResponse, error) {
			modelReq := baseReq
			modelReq.Model = route.ResolveDeployment()
			modelReq.Image = baseReq.Image
			return route.Image.Variation(callCtx, modelReq)
		},
		OverrideCost: func(metadata map[string]string) *int64 {
			return parseImageOverrideCost(metadata, imageOperationVariation)
		},
	}
}

func buildImagePricingMetadata(op imageOperationType, quality string, resolution string) map[string]string {
	metadata := map[string]string{
		"operation": imageOperationLabel(op),
	}
	q := strings.TrimSpace(quality)
	if q == "" {
		q = "standard"
	}
	if q != "" {
		metadata["quality"] = q
	}
	if res := strings.TrimSpace(resolution); res != "" {
		metadata["resolution"] = res
	}
	return metadata
}

func imageOperationLabel(op imageOperationType) string {
	switch op {
	case imageOperationEdit:
		return "image_edit"
	case imageOperationVariation:
		return "image_variation"
	default:
		return "image_generation"
	}
}
