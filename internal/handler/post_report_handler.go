package handler

import (
	"errors"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type PostReportHandler struct {
	reportService service.PostReportService
}

func NewPostReportHandler(reportService service.PostReportService) *PostReportHandler {
	return &PostReportHandler{reportService: reportService}
}

func (h *PostReportHandler) respondError(c *echo.Context, message string, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrInvalidPostID), errors.Is(err, apperrors.ErrInvalidUserID),
		errors.Is(err, apperrors.ErrInvalidReportStatus), errors.Is(err, apperrors.ErrCannotReportOwnPost):
		return response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, apperrors.ErrPostNotFound):
		return response.NotFound(c, "Post not found", err)
	case errors.Is(err, apperrors.ErrNoPendingReports), errors.Is(err, apperrors.ErrPostAlreadyHidden),
		errors.Is(err, apperrors.ErrPostNotHidden):
		return response.Conflict(c, message, err.Error())
	default:
		return response.InternalServerError(c, message, err)
	}
}

func (h *PostReportHandler) ReportPost(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	var req dto.CreatePostReportRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	if err := h.reportService.ReportPost(c.Request().Context(), c.Param("id"), userID, req); err != nil {
		return h.respondError(c, "Failed to report post", err)
	}
	// A repeat report by the same user also answers 201, so the response never
	// reveals whether the post was already reported.
	return response.Created(c, "Report submitted", nil)
}

func (h *PostReportHandler) ListReportedPosts(c *echo.Context) error {
	limit, offset := ParsePaginationParams(c, 20)
	groups, total, err := h.reportService.ListReportedPosts(c.Request().Context(), c.QueryParam("status"), limit, offset)
	if err != nil {
		return h.respondError(c, "Failed to fetch reported posts", err)
	}
	meta := response.CalculatePaginationMeta(total, offset, limit)
	return response.SuccessWithMeta(c, "Reported posts fetched successfully", groups, meta)
}

func (h *PostReportHandler) ListPostReports(c *echo.Context) error {
	limit, offset := ParsePaginationParams(c, 20)
	reports, total, err := h.reportService.ListPostReports(c.Request().Context(), c.Param("id"), c.QueryParam("status"), limit, offset)
	if err != nil {
		return h.respondError(c, "Failed to fetch post reports", err)
	}
	meta := response.CalculatePaginationMeta(total, offset, limit)
	return response.SuccessWithMeta(c, "Post reports fetched successfully", reports, meta)
}

func (h *PostReportHandler) ModeratePost(c *echo.Context) error {
	adminID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	var req dto.ModeratePostRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	if err := h.reportService.ModeratePost(c.Request().Context(), c.Param("id"), adminID, req); err != nil {
		return h.respondError(c, "Failed to moderate post", err)
	}
	return response.Success(c, "Moderation action applied", nil)
}

func (h *PostReportHandler) ListModerationActions(c *echo.Context) error {
	limit, offset := ParsePaginationParams(c, 20)
	actions, total, err := h.reportService.ListModerationActions(c.Request().Context(), c.Param("id"), limit, offset)
	if err != nil {
		return h.respondError(c, "Failed to fetch moderation history", err)
	}
	meta := response.CalculatePaginationMeta(total, offset, limit)
	return response.SuccessWithMeta(c, "Moderation history fetched successfully", actions, meta)
}
