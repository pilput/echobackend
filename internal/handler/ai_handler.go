package handler

import (
	"errors"
	"strings"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/applog"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

var aiLog = applog.Component("ai")

// MsgAIArticleUnavailable is the 503 message when AI is not configured.
const MsgAIArticleUnavailable = "AI article generation is not available"

type AIHandler struct {
	articleService service.ArticleGenerationService
}

func NewAIHandler(articleService service.ArticleGenerationService) *AIHandler {
	return &AIHandler{articleService: articleService}
}

// GenerateArticle streams an AI-drafted blog article as SSE: `ai_chunk`
// events, then exactly one of `ai_complete` ({content, model}) or `error`,
// then `data: [DONE]`. Nothing is persisted.
func (h *AIHandler) GenerateArticle(c *echo.Context) error {
	if _, ok := GetUserIDFromClaims(c); !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	var req dto.GenerateArticleRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Failed to generate article", err)
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.Title = strings.TrimSpace(req.Title)
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	if !h.articleService.Available() {
		return response.ServiceUnavailable(c, MsgAIArticleUnavailable)
	}

	chunks, complete, errCh, err := h.articleService.GenerateArticleStream(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrAIArticleGenerationUnavailable) {
			return response.ServiceUnavailable(c, MsgAIArticleUnavailable)
		}
		return response.InternalServerError(c, "Failed to generate article", err)
	}
	return streamArticleEvents(c, chunks, complete, errCh)
}

// streamArticleEvents mirrors streamChatEvents without an initial event. A
// failed write means the client is gone; returning cancels the request
// context, which stops the producer.
func streamArticleEvents(c *echo.Context, chunks <-chan string, complete <-chan dto.GeneratedArticle, errCh <-chan error) error {
	write := openAIStream(c)

	for chunk := range chunks {
		if writeSSE(write, map[string]any{"type": "ai_chunk", "data": chunk}) != nil {
			return nil //nolint:nilerr // write failed: the client is gone, nothing left to report
		}
	}

	var final map[string]any
	if streamErr, ok := <-errCh; ok && streamErr != nil {
		aiLog.Error("article stream failed", "error", streamErr)
		final = map[string]any{"type": "error", "data": streamErrorMessage(streamErr)}
	} else if article, ok := <-complete; ok {
		final = map[string]any{"type": "ai_complete", "data": article}
	}
	if final != nil && writeSSE(write, final) != nil {
		return nil //nolint:nilerr // write failed: the client is gone, nothing left to report
	}

	_ = write([]byte("data: [DONE]\n\n"))
	return nil
}
