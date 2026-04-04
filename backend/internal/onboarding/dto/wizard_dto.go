package dto

type OrganizationDetailsRequest struct {
	OrganizationName string `json:"organization_name" validate:"required,min=2,max=100"`
	Industry         string `json:"industry" validate:"required"`
	Country          string `json:"country" validate:"required,len=2"`
	City             string `json:"city,omitempty" validate:"omitempty,max=120"`
	OrganizationSize string `json:"organization_size" validate:"required"`
}

type BrandingRequest struct {
	PrimaryColor string `json:"primary_color,omitempty" validate:"omitempty,len=7"`
	AccentColor  string `json:"accent_color,omitempty" validate:"omitempty,len=7"`
	LogoFileID   string `json:"logo_file_id,omitempty" validate:"omitempty,uuid4"`
}

type TeamStepRequest struct {
	Invitations []InvitationInput `json:"invitations"`
}

type SuitesStepRequest struct {
	ActiveSuites []string `json:"active_suites" validate:"required,min=1,dive,required"`
}

type WizardStepResponse struct {
	Message          string `json:"message"`
	CurrentStep      int    `json:"current_step"`
	CompletedSteps   []int  `json:"completed_steps"`
	InvitationsSent  int    `json:"invitations_sent,omitempty"`
	OrganizationName string `json:"organization_name,omitempty"`
}
