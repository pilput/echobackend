package handler

import (
	"errors"
	"strconv"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type TagHandler struct {
	service service.TagService
}

func NewTagHandler(service service.TagService) *TagHandler {
	return &TagHandler{service: service}
}

func (h *TagHandler) CreateTag(c *echo.Context) error {
	var req dto.CreateTagRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	tag, err := h.service.CreateTag(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrTagNameRequired) {
			return response.BadRequest(c, err.Error(), nil)
		}
		return response.InternalServerError(c, "Failed to create tag", err)
	}

	return response.Created(c, "Tag created successfully", dto.TagToResponse(tag))
}

func (h *TagHandler) GetTags(c *echo.Context) error {
	tags, err := h.service.GetTags(c.Request().Context())
	if err != nil {
		return response.InternalServerError(c, "Failed to get tags", err)
	}

	tagResponses := make([]*dto.TagResponse, 0, len(tags))
	for i := range tags {
		tagResponses = append(tagResponses, dto.TagToResponse(&tags[i]))
	}

	return response.Success(c, "Successfully retrieved tags", tagResponses)
}

func (h *TagHandler) GetTrendingTags(c *echo.Context) error {
	tags, err := h.service.GetTrendingTags(c.Request().Context())
	if err != nil {
		return response.InternalServerError(c, "Failed to get trending tags", err)
	}

	return response.Success(c, "Successfully retrieved trending tags", tags)
}

func (h *TagHandler) GetTagsForSitemap(c *echo.Context) error {
	tags, err := h.service.GetTagsForSitemap(c.Request().Context(), 1000)
	if err != nil {
		return response.InternalServerError(c, "Failed to get tags for sitemap", err)
	}

	return response.Success(c, "Successfully retrieved tags for sitemap", tags)
}

func (h *TagHandler) GetTagByID(c *echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid tag ID", err)
	}

	tag, err := h.service.GetTagByID(c.Request().Context(), uint(id))
	if err != nil {
		if errors.Is(err, apperrors.ErrTagNotFound) || errors.Is(err, apperrors.ErrInvalidTagID) {
			return response.NotFound(c, "Tag not found", err)
		}
		return response.InternalServerError(c, "Failed to get tag", err)
	}

	return response.Success(c, "Successfully retrieved tag", dto.TagToResponse(tag))
}

func (h *TagHandler) UpdateTag(c *echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid tag ID", err)
	}

	var req dto.UpdateTagRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	tag, err := h.service.UpdateTag(c.Request().Context(), uint(id), &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrTagNameRequired) {
			return response.BadRequest(c, err.Error(), nil)
		}
		if errors.Is(err, apperrors.ErrTagNotFound) || errors.Is(err, apperrors.ErrInvalidTagID) {
			return response.NotFound(c, "Tag not found", err)
		}
		return response.InternalServerError(c, "Failed to update tag", err)
	}

	return response.Success(c, "Tag updated successfully", dto.TagToResponse(tag))
}

func (h *TagHandler) DeleteTag(c *echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid tag ID", err)
	}

	if err := h.service.DeleteTag(c.Request().Context(), uint(id)); err != nil {
		if errors.Is(err, apperrors.ErrTagNotFound) || errors.Is(err, apperrors.ErrInvalidTagID) {
			return response.NotFound(c, "Tag not found", err)
		}
		return response.InternalServerError(c, "Failed to delete tag", err)
	}

	return response.Success(c, "Tag deleted successfully", nil)
}
