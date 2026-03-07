package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/service"
)

// MITREHandler handles ATT&CK reference endpoints.
type MITREHandler struct {
	ruleSvc *service.RuleService
}

// NewMITREHandler creates a new MITREHandler.
func NewMITREHandler(ruleSvc *service.RuleService) *MITREHandler {
	return &MITREHandler{ruleSvc: ruleSvc}
}

func (h *MITREHandler) ListTactics(w http.ResponseWriter, r *http.Request) {
	items := mitre.AllTactics()
	out := make([]dto.MITRETacticDTO, 0, len(items))
	for _, item := range items {
		out = append(out, dto.MITRETacticDTO{
			ID:          item.ID,
			Name:        item.Name,
			ShortName:   item.ShortName,
			Description: item.Description,
		})
	}
	writeJSON(w, http.StatusOK, envelope{"data": out})
}

func (h *MITREHandler) ListTechniques(w http.ResponseWriter, r *http.Request) {
	tacticID := r.URL.Query().Get("tactic_id")
	var items []mitre.Technique
	if tacticID != "" {
		items = mitre.TechniquesByTactic(tacticID)
	} else {
		items = mitre.AllTechniques()
	}
	out := make([]dto.MITRETechniqueDTO, 0, len(items))
	for _, item := range items {
		out = append(out, dto.MITRETechniqueDTO{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			TacticIDs:   item.TacticIDs,
			Platforms:   item.Platforms,
			DataSources: item.DataSources,
		})
	}
	writeJSON(w, http.StatusOK, envelope{"data": out})
}

func (h *MITREHandler) GetTechnique(w http.ResponseWriter, r *http.Request) {
	technique, ok := mitre.TechniqueByID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "technique not found", nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": dto.MITRETechniqueDTO{
		ID:          technique.ID,
		Name:        technique.Name,
		Description: technique.Description,
		TacticIDs:   technique.TacticIDs,
		Platforms:   technique.Platforms,
		DataSources: technique.DataSources,
	}})
}

func (h *MITREHandler) Coverage(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.ruleSvc.Coverage(r.Context(), tenantID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "COVERAGE_FAILED", err.Error(), nil)
		return
	}
	out := make([]dto.MITRECoverageDTO, 0, len(items))
	for _, item := range items {
		out = append(out, dto.MITRECoverageDTO{
			TechniqueID:   item.Technique.ID,
			TechniqueName: item.Technique.Name,
			TacticIDs:     item.Technique.TacticIDs,
			HasDetection:  item.HasDetection,
			RuleCount:     item.RuleCount,
			RuleNames:     item.RuleNames,
		})
	}
	writeJSON(w, http.StatusOK, envelope{"data": out})
}
