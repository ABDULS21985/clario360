package dto

import "github.com/google/uuid"

type CreateReportRequest struct {
	Name              string      `json:"name"`
	Description       string      `json:"description"`
	ReportType        string      `json:"report_type"`
	Sections          []string    `json:"sections"`
	Period            string      `json:"period"`
	CustomPeriodStart *string     `json:"custom_period_start"`
	CustomPeriodEnd   *string     `json:"custom_period_end"`
	Schedule          *string     `json:"schedule"`
	Recipients        []uuid.UUID `json:"recipients"`
	AutoSend          bool        `json:"auto_send"`
}

type UpdateReportRequest = CreateReportRequest
