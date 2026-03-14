package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/repository"
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
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	item, err := h.ruleSvc.TechniqueDetail(r.Context(), tenantID, chi.URLParam(r, "id"), actorFromRequest(r))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, "NOT_FOUND", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
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

	// Aggregate statistics
	totalTechniques := len(items)
	coveredTechniques := 0
	activeTechniques := 0
	criticalGapCount := 0
	for _, item := range items {
		if item.HasDetection {
			coveredTechniques++
			if item.AlertCount > 0 {
				activeTechniques++
			}
		}
		if item.CoverageState == "gap" {
			criticalGapCount++
		}
	}
	passiveTechniques := coveredTechniques - activeTechniques

	coveragePercent := 0.0
	if totalTechniques > 0 {
		coveragePercent = float64(coveredTechniques) / float64(totalTechniques) * 100
	}

	// Build per-tactic coverage
	allTactics := mitre.AllTactics()
	tacticCoverage := make([]dto.MITRETacticCoverageDTO, 0, len(allTactics))
	for _, tactic := range allTactics {
		techCount := 0
		covCount := 0
		for _, item := range items {
			for _, tid := range item.TacticIDs {
				if tid == tactic.ID {
					techCount++
					if item.HasDetection {
						covCount++
					}
					break
				}
			}
		}
		tacticCoverage = append(tacticCoverage, dto.MITRETacticCoverageDTO{
			ID:             tactic.ID,
			Name:           tactic.Name,
			ShortName:      tactic.ShortName,
			TechniqueCount: techCount,
			CoveredCount:   covCount,
		})
	}

	resp := dto.MITRECoverageResponseDTO{
		Tactics:           tacticCoverage,
		Techniques:        items,
		TotalTechniques:   totalTechniques,
		CoveredTechniques: coveredTechniques,
		CoveragePercent:   coveragePercent,
		ActiveTechniques:  activeTechniques,
		PassiveTechniques: passiveTechniques,
		CriticalGapCount:  criticalGapCount,
	}

	writeJSON(w, http.StatusOK, envelope{"data": resp})
}
