package dto

// GenerateArticleRequest is the body of POST /api/ai/generate-article. The
// handler trims Prompt and Title before validation. There is deliberately no
// model or temperature field: the configured default model is always used.
type GenerateArticleRequest struct {
	Prompt string `json:"prompt" validate:"required,min=3,max=2000"`
	Tone   string `json:"tone" validate:"omitempty,oneof=informative professional casual storytelling"`
	Length string `json:"length" validate:"omitempty,oneof=short medium long"`
	Title  string `json:"title" validate:"omitempty,max=200"`
}

// GeneratedArticle is the `ai_complete` payload of the article stream.
type GeneratedArticle struct {
	Content string `json:"content"`
	Model   string `json:"model"`
}
