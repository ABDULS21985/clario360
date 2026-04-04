package dto

type ManualProvisionRequest struct {
	TenantID string `json:"tenant_id" validate:"required,uuid"`
}

type DeprovisionRequest struct {
	Reason     string `json:"reason" validate:"required,min=3,max=500"`
	RetainDays int    `json:"retain_days" validate:"required,min=1,max=3650"`
}
