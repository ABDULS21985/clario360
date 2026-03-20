package dto

type UpdateAlertStatusRequest struct {
	Status        string  `json:"status"`
	ActionNotes   *string `json:"action_notes"`
	DismissReason *string `json:"dismiss_reason"`
}
