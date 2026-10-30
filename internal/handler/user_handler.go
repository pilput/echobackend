package handler

import (
	"errors"
	"net/http"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"
	"echobackend/pkg/validator"

	"github.com/labstack/echo/v5"
)

type UserHandler struct {
	userService       service.UserService
	userFollowService service.UserFollowService
}

func NewUserHandler(userService service.UserService, userFollowService service.UserFollowService) *UserHandler {
	return &UserHandler{
		userService:       userService,
		userFollowService: userFollowService,
	}
}

func (h *UserHandler) GetByID(c *echo.Context) error {
	userID := c.Param("id")
	if !validator.IsValidUUID(userID) {
		return respondInvalidUserID(c)
	}

	if c.QueryParam("deleted") == "true" {
		userResponse, err := h.userService.GetAdminByID(c.Request().Context(), userID, true)
		if err != nil {
			return respondError(c, "Failed to retrieve user", err)
		}
		return response.Success(c, "Successfully retrieved user", userResponse)
	}

	var currentUserID string
	if uid, ok := GetUserIDFromClaims(c); ok {
		currentUserID = uid
	}

	userResponse, err := h.userFollowService.GetUserWithFollowStatus(c.Request().Context(), userID, currentUserID, true)
	if err != nil {
		return respondError(c, "Failed to retrieve user", err)
	}

	return response.Success(c, "Successfully retrieved user", userResponse)
}

func (h *UserHandler) GetByUsername(c *echo.Context) error {
	username := c.Param("username")

	var currentUserID string
	if uid, ok := GetUserIDFromClaims(c); ok {
		currentUserID = uid
	}

	user, err := h.userService.GetByUsername(c.Request().Context(), username)
	if err != nil {
		return respondError(c, "Failed to retrieve user", err)
	}

	userResponse, err := h.userFollowService.GetUserWithFollowStatus(c.Request().Context(), user.ID, currentUserID, false)
	if err != nil {
		return respondError(c, "Failed to retrieve user", err)
	}

	return response.Success(c, "Successfully retrieved user", userResponse)
}

func (h *UserHandler) GetUsers(c *echo.Context) error {
	deletedFilter, err := dto.ParseUserDeletedFilter(c.QueryParam("deleted"))
	if err != nil {
		return response.BadRequest(c, "Invalid deleted filter", err)
	}

	limit, offset := ParsePaginationParams(c, 10)

	orderBy := dto.ParseUserListOrder(c.QueryParam("order_by"))
	orderDesc := dto.ParseUserOrderDesc(c.QueryParam("order_direction"))

	users, total, err := h.userService.GetUsers(c.Request().Context(), offset, limit, deletedFilter, orderBy, orderDesc)
	if err != nil {
		return respondError(c, "Failed to retrieve users", err)
	}

	meta := response.CalculatePaginationMeta(total, offset, limit)
	return response.SuccessWithMeta(c, "Successfully retrieved users", users, meta)
}

func (h *UserHandler) DeleteUser(c *echo.Context) error {
	id := c.Param("id")
	if !validator.IsValidUUID(id) {
		return respondInvalidUserID(c)
	}
	err := h.userService.Delete(c.Request().Context(), id)
	if err != nil {
		return respondError(c, "Failed to delete user", err)
	}

	return response.Success(c, "Successfully deleted user", nil)
}

func (h *UserHandler) RestoreUser(c *echo.Context) error {
	id := c.Param("id")
	if !validator.IsValidUUID(id) {
		return respondInvalidUserID(c)
	}

	userResponse, err := h.userService.Restore(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return response.NotFound(c, "Deleted user not found", err)
		}
		if errors.Is(err, apperrors.ErrUserExists) {
			return response.Conflict(c, "Cannot restore user", "Email or username already taken by another active user")
		}
		return respondError(c, "Failed to restore user", err)
	}

	return response.Success(c, "Successfully restored user", userResponse)
}

func (h *UserHandler) GetMe(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	userResponse, err := h.userService.GetMe(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, "Failed to retrieve user", err)
	}

	return response.Success(c, "Successfully retrieved current user", userResponse)
}

// maxAvatarBodyBytes is the request-body allowance for POST /users/me/image:
// the 5 MB avatar cap plus 64 KB of multipart framing, so a file exactly at the
// cap still fits.
const maxAvatarBodyBytes = 5*1024*1024 + 64*1024

// msgAvatarTooLarge is the 413 message (and error) for an oversized avatar.
const msgAvatarTooLarge = "Avatar is too large (max 5 MB)"

func (h *UserHandler) respondAvatarError(c *echo.Context, message string, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrUserNotFound):
		return response.NotFound(c, message, err)
	case errors.Is(err, apperrors.ErrAvatarFileTooLarge):
		return response.PayloadTooLargeWithMessage(c, msgAvatarTooLarge)
	case errors.Is(err, apperrors.ErrFileNil), errors.Is(err, apperrors.ErrInvalidFileType), errors.Is(err, apperrors.ErrStorageUnavailable):
		return response.BadRequest(c, message, err)
	default:
		return respondError(c, message, err)
	}
}

func (h *UserHandler) UploadAvatar(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	req := c.Request()
	if req.ContentLength > maxAvatarBodyBytes {
		return response.PayloadTooLargeWithMessage(c, msgAvatarTooLarge)
	}
	// Bodies without a usable Content-Length (chunked) are capped while the
	// multipart form is parsed.
	req.Body = http.MaxBytesReader(c.Response(), req.Body, maxAvatarBodyBytes)

	file, err := c.FormFile("avatar")
	if err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			return response.PayloadTooLargeWithMessage(c, msgAvatarTooLarge)
		}
		return response.BadRequest(c, "Failed to upload image", err)
	}
	if file == nil {
		return response.BadRequest(c, "No file uploaded", nil)
	}

	imagePath, err := h.userService.UploadAvatar(c.Request().Context(), userID, file)
	if err != nil {
		return h.respondAvatarError(c, "Failed to upload image", err)
	}

	return response.Success(c, "Profile picture updated successfully", map[string]any{
		"image": imagePath,
	})
}

func (h *UserHandler) CreateUser(c *echo.Context) error {
	var req dto.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request format", err)
	}

	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	userResponse, err := h.userService.CreateUser(c.Request().Context(), &req)
	if errors.Is(err, apperrors.ErrUserExists) {
		return response.Conflict(c, "Failed to create user", "Email or username already exists")
	}
	if err != nil {
		return respondError(c, "Failed to create user", err)
	}

	return response.Created(c, "User created successfully", userResponse)
}

func (h *UserHandler) UpdateUser(c *echo.Context) error {
	id := c.Param("id")
	if !validator.IsValidUUID(id) {
		return respondInvalidUserID(c)
	}

	var req dto.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request format", err)
	}

	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	userResponse, err := h.userService.UpdateUser(c.Request().Context(), id, &req)
	if errors.Is(err, apperrors.ErrUserNotFound) {
		return response.NotFound(c, "Failed to update user", err)
	}
	if errors.Is(err, apperrors.ErrUserExists) {
		return response.Conflict(c, "Failed to update user", "Email or username already taken")
	}
	if err != nil {
		return respondError(c, "Failed to update user", err)
	}

	return response.Success(c, "User updated successfully", userResponse)
}
