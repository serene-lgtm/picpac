package response

// SendPhoneCodeResponse defines the API response for sending a phone code.
type SendPhoneCodeResponse struct {
	Sent bool `json:"sent"`
}

// AuthResponse defines the API response for successful authentication.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// RefreshAccessTokenResponse defines the API response for refreshing an access token.
type RefreshAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// UserResponse defines the API response for a user.
type UserResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Status      string `json:"status"`
}

// LogoutResponse defines the API response for logout.
type LogoutResponse struct {
	LoggedOut bool `json:"logged_out"`
}
