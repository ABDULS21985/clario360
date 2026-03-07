package dto

import "strings"

type CreateMinutesRequest struct {
	Content string `json:"content"`
}

type UpdateMinutesRequest struct {
	Content string `json:"content"`
}

type ReviewRequest struct {
	Notes string `json:"notes"`
}

func (r *CreateMinutesRequest) Normalize() {
	r.Content = strings.TrimSpace(r.Content)
}

func (r *UpdateMinutesRequest) Normalize() {
	r.Content = strings.TrimSpace(r.Content)
}

func (r *ReviewRequest) Normalize() {
	r.Notes = strings.TrimSpace(r.Notes)
}
