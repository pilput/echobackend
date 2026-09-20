package handler

import (
	"errors"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

type GuildHandler struct {
	guildService service.GuildService
}

func NewGuildHandler(guildService service.GuildService) *GuildHandler {
	return &GuildHandler{guildService: guildService}
}

func (h *GuildHandler) CreateGuild(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	var req dto.CreateGuildRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	guild, err := h.guildService.CreateGuild(c.Request().Context(), userID, &req)
	if err != nil {
		if handled := handleGuildError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to create guild", err)
	}
	return response.Created(c, "Guild created successfully", guild)
}

// ListGuilds returns the public guild directory. It is open to anonymous
// callers, so no viewer-relative fields are filled in.
func (h *GuildHandler) ListGuilds(c *echo.Context) error {
	limit, offset := ParsePaginationParams(c, 20)
	search := c.QueryParam("search")

	guilds, total, err := h.guildService.ListGuilds(c.Request().Context(), search, limit, offset)
	if err != nil {
		if handled := handleGuildError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to fetch guilds", err)
	}

	meta := response.CalculatePaginationMeta(total, offset, limit)
	return response.SuccessWithMeta(c, "Guilds fetched successfully", guilds, meta)
}

// GetMyGuilds lists the guilds the caller belongs to, including private ones.
func (h *GuildHandler) GetMyGuilds(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	limit, offset := ParsePaginationParams(c, 20)

	guilds, total, err := h.guildService.ListMyGuilds(c.Request().Context(), userID, limit, offset)
	if err != nil {
		if handled := handleGuildError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to fetch guilds", err)
	}

	meta := response.CalculatePaginationMeta(total, offset, limit)
	return response.SuccessWithMeta(c, "Guilds fetched successfully", guilds, meta)
}

func (h *GuildHandler) GetGuild(c *echo.Context) error {
	guildSlug := c.Param("slug")
	if guildSlug == "" {
		return response.BadRequest(c, "Guild slug is required", nil)
	}
	// OptionalAuth: an anonymous caller simply gets no membership fields.
	viewerID, _ := GetUserIDFromClaims(c)

	guild, err := h.guildService.GetGuildBySlug(c.Request().Context(), guildSlug, viewerID)
	if err != nil {
		if handled := handleGuildError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to fetch guild", err)
	}
	return response.Success(c, "Guild fetched successfully", guild)
}

func (h *GuildHandler) UpdateGuild(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	guildSlug := c.Param("slug")
	if guildSlug == "" {
		return response.BadRequest(c, "Guild slug is required", nil)
	}

	var req dto.UpdateGuildRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	guild, err := h.guildService.UpdateGuild(c.Request().Context(), guildSlug, userID, &req)
	if err != nil {
		if handled := handleGuildError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to update guild", err)
	}
	return response.Success(c, "Guild updated successfully", guild)
}

func (h *GuildHandler) DeleteGuild(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	guildSlug := c.Param("slug")
	if guildSlug == "" {
		return response.BadRequest(c, "Guild slug is required", nil)
	}

	if err := h.guildService.DeleteGuild(c.Request().Context(), guildSlug, userID); err != nil {
		if handled := handleGuildError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to delete guild", err)
	}
	return response.Success(c, "Guild deleted successfully", nil)
}

func (h *GuildHandler) JoinGuild(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	guildSlug := c.Param("slug")
	if guildSlug == "" {
		return response.BadRequest(c, "Guild slug is required", nil)
	}

	member, err := h.guildService.JoinGuild(c.Request().Context(), guildSlug, userID)
	if err != nil {
		if handled := handleGuildError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to join guild", err)
	}
	return response.Created(c, "Joined guild successfully", member)
}

func (h *GuildHandler) LeaveGuild(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	guildSlug := c.Param("slug")
	if guildSlug == "" {
		return response.BadRequest(c, "Guild slug is required", nil)
	}

	if err := h.guildService.LeaveGuild(c.Request().Context(), guildSlug, userID); err != nil {
		if handled := handleGuildError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to leave guild", err)
	}
	return response.Success(c, "Left guild successfully", nil)
}

func (h *GuildHandler) GetMembers(c *echo.Context) error {
	guildSlug := c.Param("slug")
	if guildSlug == "" {
		return response.BadRequest(c, "Guild slug is required", nil)
	}
	viewerID, _ := GetUserIDFromClaims(c)
	limit, offset := ParsePaginationParams(c, 50)

	members, total, err := h.guildService.ListMembers(c.Request().Context(), guildSlug, viewerID, limit, offset)
	if err != nil {
		if handled := handleGuildError(c, err); handled != nil {
			return handled
		}
		return response.InternalServerError(c, "Failed to fetch guild members", err)
	}

	meta := response.CalculatePaginationMeta(total, offset, limit)
	return response.SuccessWithMeta(c, "Guild members fetched successfully", members, meta)
}

// handleGuildError maps guild domain errors to responses, returning nil when
// the error is not one of them so the caller can fall back to a 500.
func handleGuildError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrGuildNotFound):
		return response.NotFound(c, "Guild not found", err)
	case errors.Is(err, apperrors.ErrGuildNotOwned):
		return response.Forbidden(c, "Access forbidden")
	case errors.Is(err, apperrors.ErrGuildSlugExists):
		return response.Conflict(c, "Guild slug already taken", err.Error())
	case errors.Is(err, apperrors.ErrAlreadyGuildMember):
		return response.Conflict(c, "Already a member of this guild", err.Error())
	case errors.Is(err, apperrors.ErrNotGuildMember),
		errors.Is(err, apperrors.ErrGuildOwnerCannotLeave),
		errors.Is(err, apperrors.ErrGuildSlugInvalid),
		errors.Is(err, apperrors.ErrGuildSlugReserved):
		return response.BadRequest(c, err.Error(), err)
	default:
		return nil
	}
}
