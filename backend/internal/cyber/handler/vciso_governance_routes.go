package handler

import "github.com/go-chi/chi/v5"

// RegisterVCISOGovernanceRoutes registers all vCISO governance routes directly
// on the router using full paths. This avoids chi's "duplicate mount" panic
// since the vCISO chat handler already mounts a sub-router at /api/v1/cyber/vciso.
func RegisterVCISOGovernanceRoutes(r chi.Router, h *VCISOGovernanceHandler) {
	if h == nil {
		return
	}

	const p = "/api/v1/cyber/vciso"

	// ── Risks ──────────────────────────────────────────────
	r.Get(p+"/risks/stats", h.RiskStats)
	r.Get(p+"/risks", h.ListRisks)
	r.Post(p+"/risks", h.CreateRisk)
	r.Get(p+"/risks/{id}", h.GetRisk)
	r.Put(p+"/risks/{id}", h.UpdateRisk)
	r.Delete(p+"/risks/{id}", h.DeleteRisk)

	// ── Policies ──────────────────────────────────────────
	r.Get(p+"/policies/stats", h.PolicyStats)
	r.Post(p+"/policies/generate", h.GeneratePolicy)
	r.Get(p+"/policies", h.ListPolicies)
	r.Post(p+"/policies", h.CreatePolicy)
	r.Get(p+"/policies/{id}", h.GetPolicy)
	r.Put(p+"/policies/{id}", h.UpdatePolicy)
	r.Delete(p+"/policies/{id}", h.DeletePolicy)
	r.Put(p+"/policies/{id}/status", h.UpdatePolicyStatus)

	// ── Policy Exceptions ─────────────────────────────────
	r.Get(p+"/policy-exceptions", h.ListPolicyExceptions)
	r.Post(p+"/policy-exceptions", h.CreatePolicyException)
	r.Put(p+"/policy-exceptions/{id}/decision", h.DecidePolicyException)

	// ── Vendors ───────────────────────────────────────────
	r.Get(p+"/vendors/stats", h.VendorStats)
	r.Get(p+"/vendors", h.ListVendors)
	r.Post(p+"/vendors", h.CreateVendor)
	r.Get(p+"/vendors/{id}", h.GetVendor)
	r.Put(p+"/vendors/{id}", h.UpdateVendor)
	r.Delete(p+"/vendors/{id}", h.DeleteVendor)
	r.Put(p+"/vendors/{id}/status", h.UpdateVendorStatus)

	// ── Questionnaires ────────────────────────────────────
	r.Get(p+"/questionnaires", h.ListQuestionnaires)
	r.Post(p+"/questionnaires", h.CreateQuestionnaire)
	r.Put(p+"/questionnaires/{id}", h.UpdateQuestionnaire)
	r.Put(p+"/questionnaires/{id}/status", h.UpdateQuestionnaireStatus)

	// ── Evidence ──────────────────────────────────────────
	r.Get(p+"/evidence/stats", h.EvidenceStats)
	r.Get(p+"/evidence", h.ListEvidence)
	r.Post(p+"/evidence", h.CreateEvidence)
	r.Get(p+"/evidence/{id}", h.GetEvidence)
	r.Put(p+"/evidence/{id}", h.UpdateEvidence)
	r.Delete(p+"/evidence/{id}", h.DeleteEvidence)
	r.Post(p+"/evidence/{id}/verify", h.VerifyEvidence)

	// ── Maturity ──────────────────────────────────────────
	r.Get(p+"/maturity", h.ListMaturityAssessments)
	r.Post(p+"/maturity", h.CreateMaturityAssessment)

	// ── Benchmarks ────────────────────────────────────────
	r.Get(p+"/benchmarks", h.ListBenchmarks)

	// ── Budget ────────────────────────────────────────────
	r.Get(p+"/budget/summary", h.BudgetSummary)
	r.Get(p+"/budget", h.ListBudgetItems)
	r.Post(p+"/budget", h.CreateBudgetItem)
	r.Put(p+"/budget/{id}", h.UpdateBudgetItem)
	r.Delete(p+"/budget/{id}", h.DeleteBudgetItem)

	// ── Awareness ─────────────────────────────────────────
	r.Get(p+"/awareness", h.ListAwarenessPrograms)
	r.Post(p+"/awareness", h.CreateAwarenessProgram)
	r.Put(p+"/awareness/{id}", h.UpdateAwarenessProgram)

	// ── IAM Findings ──────────────────────────────────────
	r.Get(p+"/iam-findings/summary", h.IAMFindingSummary)
	r.Get(p+"/iam-findings", h.ListIAMFindings)
	r.Put(p+"/iam-findings/{id}", h.UpdateIAMFinding)

	// ── Escalation Rules ──────────────────────────────────
	r.Get(p+"/escalation-rules", h.ListEscalationRules)
	r.Post(p+"/escalation-rules", h.CreateEscalationRule)
	r.Put(p+"/escalation-rules/{id}", h.UpdateEscalationRule)
	r.Delete(p+"/escalation-rules/{id}", h.DeleteEscalationRule)

	// ── Playbooks ─────────────────────────────────────────
	r.Get(p+"/playbooks", h.ListPlaybooks)
	r.Post(p+"/playbooks", h.CreatePlaybook)
	r.Put(p+"/playbooks/{id}", h.UpdatePlaybook)
	r.Delete(p+"/playbooks/{id}", h.DeletePlaybook)
	r.Post(p+"/playbooks/{id}/simulate", h.SimulatePlaybook)

	// ── Obligations ───────────────────────────────────────
	r.Get(p+"/obligations", h.ListObligations)
	r.Post(p+"/obligations", h.CreateObligation)
	r.Put(p+"/obligations/{id}", h.UpdateObligation)
	r.Delete(p+"/obligations/{id}", h.DeleteObligation)

	// ── Control Tests ─────────────────────────────────────
	r.Get(p+"/control-tests", h.ListControlTests)
	r.Post(p+"/control-tests", h.CreateControlTest)

	// ── Control Dependencies ──────────────────────────────
	r.Get(p+"/control-dependencies", h.ListControlDependencies)

	// ── Integrations ──────────────────────────────────────
	r.Get(p+"/integrations", h.ListIntegrations)
	r.Post(p+"/integrations", h.CreateIntegration)
	r.Put(p+"/integrations/{id}", h.UpdateIntegration)
	r.Patch(p+"/integrations/{id}", h.PatchIntegration)
	r.Delete(p+"/integrations/{id}", h.DeleteIntegration)
	r.Post(p+"/integrations/{id}/sync", h.SyncIntegration)

	// ── Control Ownership ─────────────────────────────────
	r.Get(p+"/control-ownership", h.ListControlOwnerships)
	r.Post(p+"/control-ownership", h.CreateControlOwnership)
	r.Put(p+"/control-ownership/{id}", h.UpdateControlOwnership)
	r.Post(p+"/control-ownership/{id}/mark-reviewed", h.MarkControlOwnershipReviewed)

	// ── Approvals ─────────────────────────────────────────
	r.Get(p+"/approvals", h.ListApprovals)
	r.Post(p+"/approvals", h.CreateApproval)
	r.Put(p+"/approvals/{id}/decision", h.DecideApproval)
}
