package models

// ModerationRequest is a normalized moderation invocation sent to providers.
type ModerationRequest struct {
	Model string
	Input []string
}

// ModerationResponse captures OpenAI-style moderation results.
type ModerationResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Results []ModerationResult `json:"results"`
	Usage   Usage              `json:"usage"`
}

// ModerationResult is a single moderation outcome for the provided inputs.
type ModerationResult struct {
	Categories                ModerationCategories                `json:"categories"`
	CategoryAppliedInputTypes ModerationCategoryAppliedInputTypes `json:"category_applied_input_types"`
	CategoryScores            ModerationCategoryScores            `json:"category_scores"`
	Flagged                   bool                                `json:"flagged"`
}

// ModerationCategories mirrors OpenAI category booleans.
type ModerationCategories struct {
	Harassment            bool `json:"harassment"`
	HarassmentThreatening bool `json:"harassment/threatening"`
	Hate                  bool `json:"hate"`
	HateThreatening       bool `json:"hate/threatening"`
	Illicit               bool `json:"illicit"`
	IllicitViolent        bool `json:"illicit/violent"`
	SelfHarm              bool `json:"self-harm"`
	SelfHarmInstructions  bool `json:"self-harm/instructions"`
	SelfHarmIntent        bool `json:"self-harm/intent"`
	Sexual                bool `json:"sexual"`
	SexualMinors          bool `json:"sexual/minors"`
	Violence              bool `json:"violence"`
	ViolenceGraphic       bool `json:"violence/graphic"`
}

// ModerationCategoryAppliedInputTypes lists the modalities per category.
type ModerationCategoryAppliedInputTypes struct {
	Harassment            []string `json:"harassment"`
	HarassmentThreatening []string `json:"harassment/threatening"`
	Hate                  []string `json:"hate"`
	HateThreatening       []string `json:"hate/threatening"`
	Illicit               []string `json:"illicit"`
	IllicitViolent        []string `json:"illicit/violent"`
	SelfHarm              []string `json:"self-harm"`
	SelfHarmInstructions  []string `json:"self-harm/instructions"`
	SelfHarmIntent        []string `json:"self-harm/intent"`
	Sexual                []string `json:"sexual"`
	SexualMinors          []string `json:"sexual/minors"`
	Violence              []string `json:"violence"`
	ViolenceGraphic       []string `json:"violence/graphic"`
}

// ModerationCategoryScores maps categories to their likelihood scores.
type ModerationCategoryScores struct {
	Harassment            float64 `json:"harassment"`
	HarassmentThreatening float64 `json:"harassment/threatening"`
	Hate                  float64 `json:"hate"`
	HateThreatening       float64 `json:"hate/threatening"`
	Illicit               float64 `json:"illicit"`
	IllicitViolent        float64 `json:"illicit/violent"`
	SelfHarm              float64 `json:"self-harm"`
	SelfHarmInstructions  float64 `json:"self-harm/instructions"`
	SelfHarmIntent        float64 `json:"self-harm/intent"`
	Sexual                float64 `json:"sexual"`
	SexualMinors          float64 `json:"sexual/minors"`
	Violence              float64 `json:"violence"`
	ViolenceGraphic       float64 `json:"violence/graphic"`
}
