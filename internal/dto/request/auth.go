package request

// SendPhoneCodeInput defines the input for sending a phone login code.
type SendPhoneCodeInput struct {
	Phone string `json:"phone"`
}

// PhoneLoginInput defines the input for phone login.
type PhoneLoginInput struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// RefreshTokenInput defines the input for refreshing auth tokens.
type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutInput defines the input for logging out.
type LogoutInput struct {
	RefreshToken string `json:"refresh_token"`
}
