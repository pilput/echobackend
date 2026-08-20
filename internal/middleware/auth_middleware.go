package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"echobackend/config"
	"echobackend/internal/service"
	"echobackend/pkg/response"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// AuthMiddleware provides authentication middleware for Echo
type AuthMiddleware struct {
	conf        *config.Config
	userService service.UserService
}

// NewAuthMiddleware creates a new instance of AuthMiddleware
func NewAuthMiddleware(conf *config.Config, userService service.UserService) *AuthMiddleware {
	return &AuthMiddleware{
		conf:        conf,
		userService: userService,
	}
}

// Auth validates JWT tokens and sets user claims in the context
func (a *AuthMiddleware) Auth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				log.Warn("auth: missing authorization header", "path", c.Request().URL.Path, "remote_ip", c.RealIP())
				return response.Unauthorized(c, "Missing authorization header")
			}

			tokenString, err := extractBearerToken(authHeader)
			if err != nil {
				log.Warn("auth: malformed authorization header", "path", c.Request().URL.Path, "remote_ip", c.RealIP(), "error", err)
				return response.Unauthorized(c, "Invalid authorization header")
			}

			claims, err := validateToken(tokenString, a.conf.Auth.JWTSecret)
			if err != nil {
				// Log the real parse/validation error server-side only; never expose it to clients.
				log.Warn("auth: invalid token", "path", c.Request().URL.Path, "remote_ip", c.RealIP(), "error", err)
				return response.Unauthorized(c, "Invalid or expired token")
			}

			c.Set("user", claims)
			return next(c)
		}
	}
}

// OptionalAuth validates JWT tokens if present but does not require them
func (a *AuthMiddleware) OptionalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return next(c)
			}

			tokenString, err := extractBearerToken(authHeader)
			if err != nil {
				return next(c)
			}

			claims, err := validateToken(tokenString, a.conf.Auth.JWTSecret)
			if err != nil {
				return next(c)
			}

			c.Set("user", claims)
			return next(c)
		}
	}
}

// AuthAdmin validates that the user has admin privileges
func (a *AuthMiddleware) AuthAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			userClaims := c.Get("user")
			if userClaims == nil {
				return response.Unauthorized(c, "Authentication required")
			}

			claims, ok := userClaims.(jwt.MapClaims)
			if !ok {
				return response.Unauthorized(c, "Authentication required")
			}

			// Fast-path: check is_super_admin from JWT claims if present
			if isSuperAdminClaim, exists := claims["is_super_admin"]; exists && isSuperAdminClaim != nil {
				if isSuperAdmin, ok := isSuperAdminClaim.(bool); ok && isSuperAdmin {
					return next(c)
				}
			}

			userID, err := getUserIDFromClaims(claims)
			if err != nil {
				log.Warn("auth: admin check failed to resolve user id", "path", c.Request().URL.Path, "remote_ip", c.RealIP(), "error", err)
				return response.Unauthorized(c, "Authentication required")
			}

			isSuperAdmin, err := a.isSuperAdminFromDB(c.Request().Context(), userID)
			if err != nil {
				log.Warn("auth: failed to validate admin privileges", "path", c.Request().URL.Path, "remote_ip", c.RealIP(), "user_id", userID, "error", err)
				return response.Unauthorized(c, "Failed to validate privileges")
			}

			if !isSuperAdmin {
				log.Warn("auth: insufficient privileges", "path", c.Request().URL.Path, "remote_ip", c.RealIP(), "user_id", userID)
				return response.Forbidden(c, "Insufficient privileges")
			}

			return next(c)
		}
	}
}

// extractBearerToken extracts the token from the Authorization header
func extractBearerToken(authHeader string) (string, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid token format, expected 'Bearer <token>'")
	}
	return parts[1], nil
}

// validateToken validates the JWT token and returns the claims
func validateToken(tokenString, jwtSecret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

func getUserIDFromClaims(claims jwt.MapClaims) (string, error) {
	userID, exists := claims["user_id"]
	if !exists {
		return "", errors.New("unauthorized: user ID not found in token")
	}

	switch v := userID.(type) {
	case string:
		if v == "" {
			return "", errors.New("unauthorized: invalid user ID in token")
		}
		return v, nil
	default:
		return "", errors.New("unauthorized: invalid user ID format in token")
	}
}

func (a *AuthMiddleware) isSuperAdminFromDB(ctx context.Context, userID string) (bool, error) {
	user, err := a.userService.GetAdminByID(ctx, userID, false)
	if err != nil {
		return false, err
	}

	if user.IsSuperAdmin == nil {
		return false, nil
	}

	return *user.IsSuperAdmin, nil
}
