package dto

import (
	"strings"

	"github.com/clario360/platform/internal/lex/model"
)

type UpdateClauseReviewRequest struct {
	Status model.ClauseReviewStatus `json:"status"`
	Notes  string                   `json:"notes"`
}

func (r *UpdateClauseReviewRequest) Normalize() {
	r.Notes = strings.TrimSpace(r.Notes)
}
