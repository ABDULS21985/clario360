package dto

type RegisterRequest struct {
	OrganizationName string `json:"organization_name" validate:"required,min=2,max=100"`
	AdminEmail       string `json:"admin_email" validate:"required,email,max=255"`
	AdminFirstName   string `json:"admin_first_name" validate:"required,min=1,max=100"`
	AdminLastName    string `json:"admin_last_name" validate:"required,min=1,max=100"`
	AdminPassword    string `json:"admin_password" validate:"required,min=12,max=128"`
	Country          string `json:"country" validate:"required,len=2"`
	Industry         string `json:"industry" validate:"required"`
	ReferralSource   string `json:"referral_source,omitempty" validate:"omitempty,max=255"`
	CaptchaToken     string `json:"captcha_token,omitempty" validate:"omitempty,max=2048"`
}

type RegisterResponse struct {
	TenantID        string `json:"tenant_id,omitempty"`
	Email           string `json:"email"`
	Message         string `json:"message"`
	VerificationTTL int    `json:"verification_ttl_seconds,omitempty"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
	OTP   string `json:"otp" validate:"required,len=6,numeric"`
}

type VerifyEmailResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresAt    string `json:"expires_at"`
	TenantID     string `json:"tenant_id"`
	Message      string `json:"message"`
}

type ResendOTPRequest struct {
	Email        string `json:"email" validate:"required,email,max=255"`
	CaptchaToken string `json:"captcha_token,omitempty" validate:"omitempty,max=2048"`
}
