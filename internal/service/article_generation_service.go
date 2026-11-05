package service

import (
	"context"
	"strconv"
	"strings"

	"echobackend/config"
	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/platform/openrouter"
)

// ArticleGenerationService drafts a blog article with the configured
// OpenRouter model. It is stateless: nothing is read from or written to the DB.
type ArticleGenerationService interface {
	// Available reports whether an article can be generated: an OpenRouter
	// client is wired, OPENROUTER_API_KEY is non-blank and
	// OPENROUTER_DEFAULT_MODEL is set.
	Available() bool
	// GenerateArticleStream streams the article. The chunk channel delivers
	// text deltas; afterwards exactly one of complete or errCh carries a value,
	// then all three channels close. When the service is not Available it
	// returns apperrors.ErrAIArticleGenerationUnavailable and nil channels without
	// calling OpenRouter.
	GenerateArticleStream(ctx context.Context, req *dto.GenerateArticleRequest) (<-chan string, <-chan dto.GeneratedArticle, <-chan error, error)
}

// articleTemperature is fixed: the endpoint exposes no temperature control.
const articleTemperature = 0.7

const (
	defaultArticleTone   = "informative"
	defaultArticleLength = "medium"
)

const articleSystemPrompt = `You are an expert blog writer for pilput, a publishing platform for writers. Write a complete, original, well-structured blog article based on the user's request.

Output rules:
- Respond with HTML only. Do not use Markdown and do not wrap the answer in code fences.
- Do not include <html>, <head>, <body>, <style> or <script> tags, images, or inline styles.
- Start with the article title in a single <h1> element.
- Write the body using only these tags: <h2>, <h3>, <p>, <ul>, <ol>, <li>, <strong>, <em>, <blockquote>, <code>, <pre>, <a href="...">.
- Write in the same language as the user's request.
- Do not add any commentary before or after the article.`

var articleToneDescriptions = map[string]string{
	"informative":  "informative and clear",
	"professional": "professional and authoritative",
	"casual":       "casual and friendly",
	"storytelling": "narrative, storytelling style",
}

var articleLengthWords = map[string]int{
	"short":  400,
	"medium": 800,
	"long":   1500,
}

type articleGenerationService struct {
	openRouter openrouter.Client
	config     *config.Config
}

func NewArticleGenerationService(openRouter openrouter.Client, cfg *config.Config) ArticleGenerationService {
	return &articleGenerationService{openRouter: openRouter, config: cfg}
}

func (s *articleGenerationService) Available() bool {
	if s.openRouter == nil || s.config == nil {
		return false
	}
	return strings.TrimSpace(s.config.OpenRouter.APIKey) != "" && s.defaultModel() != ""
}

func (s *articleGenerationService) defaultModel() string {
	if s.config == nil {
		return ""
	}
	return strings.TrimSpace(s.config.OpenRouter.DefaultModel)
}

func (s *articleGenerationService) GenerateArticleStream(ctx context.Context, req *dto.GenerateArticleRequest) (<-chan string, <-chan dto.GeneratedArticle, <-chan error, error) {
	if !s.Available() {
		return nil, nil, nil, apperrors.ErrAIArticleGenerationUnavailable
	}

	model := s.defaultModel()
	upstreamChunks, usageCh, upstreamErrCh := s.openRouter.GenerateStream(ctx, buildArticleMessages(req), &model, articleTemperature)
	outChunks := make(chan string)
	complete := make(chan dto.GeneratedArticle, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(outChunks)
		defer close(complete)
		defer close(errCh)

		var content strings.Builder
		for chunk := range upstreamChunks {
			content.WriteString(chunk)
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case outChunks <- chunk:
			}
		}

		if err, ok := <-upstreamErrCh; ok && err != nil {
			errCh <- err
			return
		}
		// Drain usage so the client goroutine is never left blocked; it is
		// not reported for article drafts.
		if usageCh != nil {
			<-usageCh
		}
		if content.Len() == 0 {
			errCh <- apperrors.ErrAIEmptyResponse
			return
		}
		complete <- dto.GeneratedArticle{Content: content.String(), Model: model}
	}()

	return outChunks, complete, errCh, nil
}

// buildArticleMessages returns the system + user messages sent to OpenRouter.
func buildArticleMessages(req *dto.GenerateArticleRequest) []openrouter.Message {
	return []openrouter.Message{
		{Role: "system", Content: articleSystemPrompt},
		{Role: "user", Content: buildArticleUserPrompt(req)},
	}
}

// buildArticleUserPrompt renders the user message; unknown or blank tone and
// length fall back to the defaults, and a blank title is left out.
func buildArticleUserPrompt(req *dto.GenerateArticleRequest) string {
	tone, ok := articleToneDescriptions[strings.TrimSpace(req.Tone)]
	if !ok {
		tone = articleToneDescriptions[defaultArticleTone]
	}
	words, ok := articleLengthWords[strings.TrimSpace(req.Length)]
	if !ok {
		words = articleLengthWords[defaultArticleLength]
	}

	lines := []string{
		"Write a blog article about: " + strings.TrimSpace(req.Prompt),
		"Tone: " + tone,
		"Length: about " + strconv.Itoa(words) + " words",
	}
	if title := strings.TrimSpace(req.Title); title != "" {
		lines = append(lines, "Working title: "+title)
	}
	return strings.Join(lines, "\n")
}
