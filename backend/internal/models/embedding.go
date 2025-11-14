package models

type EmbeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type Embedding struct {
	Index  int       `json:"index"`
	Vector []float32 `json:"vector"`
}

type EmbeddingsResponse struct {
	Model      string      `json:"model"`
	Embeddings []Embedding `json:"data"`
	Usage      Usage       `json:"usage"`
}
