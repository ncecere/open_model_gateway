package batchworker

import (
	"context"

	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
)

func newImageGenerationOperationConfig(req models.ImageRequest) executor.ImageOperationConfig {
	baseReq := req
	return executor.ImageOperationConfig{
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
	return executor.ImageOperationConfig{
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
	return executor.ImageOperationConfig{
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
