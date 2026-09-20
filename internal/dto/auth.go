package dto

type LoginRequest struct {
	Identifier string `json:"identifier" validate:"required,max=255"`
	Password   string `json:"password" validate:"required,min=6"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Username string `json:"username" validate:"required,min=3,max=30"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=8"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RevokeAllSessionsRequest guards the platform-wide logout behind an explicit
// acknowledgement, so the endpoint cannot be triggered by a stray request.
type RevokeAllSessionsRequest struct {
	Confirm bool `json:"confirm"`
}

type OAuthExchangeRequest struct {
	Code string `json:"code" validate:"required"`
}

type UpdateProfileRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=30"`
	FirstName string `json:"first_name" validate:"omitempty,max=100"`
	LastName  string `json:"last_name" validate:"omitempty,max=100"`
}

type CheckUsernameRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30"`
}

type AuthTokenResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	User         *UserBrief `json:"user"`
}
