//go:build integration

package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/lex/dto"
	"github.com/clario360/platform/internal/lex/model"
)

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func fileReference(fileName, text, changeSummary string) dto.FileReference {
	return dto.FileReference{
		FileID:        uuid.New(),
		FileName:      fileName,
		FileSizeBytes: int64(len(text)),
		ContentHash:   hashText(text),
		ExtractedText: text,
		ChangeSummary: changeSummary,
	}
}

func (h *lexHarness) baseContractRequest(title string, contractType model.ContractType, totalValue float64, documentText string) dto.CreateContractRequest {
	effectiveDate := time.Now().UTC().AddDate(0, -1, 0)
	expiryDate := time.Now().UTC().AddDate(0, 6, 0)
	paymentTerms := "net_30"
	department := "Legal Operations"
	ownerName := "Integration Owner"
	reviewerID := uuid.New()
	reviewerName := "Integration Reviewer"
	req := dto.CreateContractRequest{
		Title:             title,
		Type:              contractType,
		Description:       title + " created by the Lex integration suite.",
		PartyAName:        "Clario Holdings Limited",
		PartyBName:        "Counterparty " + title,
		TotalValue:        &totalValue,
		Currency:          "SAR",
		PaymentTerms:      &paymentTerms,
		EffectiveDate:     &effectiveDate,
		ExpiryDate:        &expiryDate,
		AutoRenew:         false,
		RenewalNoticeDays: 30,
		OwnerUserID:       h.userID,
		OwnerName:         ownerName,
		LegalReviewerID:   &reviewerID,
		LegalReviewerName: &reviewerName,
		Department:        &department,
		Tags:              []string{"integration", strings.ToLower(string(contractType))},
		Metadata:          map[string]any{"source": "integration-test"},
	}
	if strings.TrimSpace(documentText) != "" {
		fileName := strings.NewReplacer(" ", "-", "/", "-", "(", "", ")", "").Replace(strings.ToLower(title)) + ".txt"
		ref := fileReference(fileName, documentText, "Initial integration document.")
		req.Document = &ref
	}
	return req
}

func (h *lexHarness) createContract(t *testing.T, req dto.CreateContractRequest) model.Contract {
	t.Helper()
	return mustData[model.Contract](t, h.doJSON(t, http.MethodPost, "/api/v1/lex/contracts", req), http.StatusCreated)
}

func (h *lexHarness) createContractWithText(t *testing.T, title string, contractType model.ContractType, totalValue float64, documentText string) model.Contract {
	t.Helper()
	return h.createContract(t, h.baseContractRequest(title, contractType, totalValue, documentText))
}

func (h *lexHarness) uploadContractDocument(t *testing.T, contractID uuid.UUID, fileName, text, changeSummary string) []model.ContractVersion {
	t.Helper()
	return mustData[[]model.ContractVersion](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/lex/contracts/%s/upload", contractID), dto.UploadContractDocumentRequest{
		FileReference: fileReference(fileName, text, changeSummary),
	}), http.StatusOK)
}

func (h *lexHarness) analyzeContract(t *testing.T, contractID uuid.UUID) model.AnalysisResult {
	t.Helper()
	return mustData[model.AnalysisResult](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/lex/contracts/%s/analyze", contractID), nil), http.StatusOK)
}

func (h *lexHarness) updateContractStatus(t *testing.T, contractID uuid.UUID, status model.ContractStatus) model.Contract {
	t.Helper()
	return mustData[model.Contract](t, h.doJSON(t, http.MethodPut, fmt.Sprintf("/api/v1/lex/contracts/%s/status", contractID), dto.UpdateContractStatusRequest{
		Status: status,
	}), http.StatusOK)
}

func (h *lexHarness) activateContract(t *testing.T, contractID uuid.UUID) model.Contract {
	t.Helper()
	h.updateContractStatus(t, contractID, model.ContractStatusInternalReview)
	h.updateContractStatus(t, contractID, model.ContractStatusLegalReview)
	h.updateContractStatus(t, contractID, model.ContractStatusNegotiation)
	h.updateContractStatus(t, contractID, model.ContractStatusPendingSignature)
	return h.updateContractStatus(t, contractID, model.ContractStatusActive)
}

func (h *lexHarness) startReview(t *testing.T, contractID uuid.UUID, description string) model.LegalWorkflowSummary {
	t.Helper()
	return mustData[model.LegalWorkflowSummary](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/lex/contracts/%s/review", contractID), dto.ReviewContractRequest{
		Description: description,
		SLAHours:    48,
	}), http.StatusAccepted)
}

func (h *lexHarness) scalarInt(t *testing.T, query string, args ...any) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var count int
	if err := h.env.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		t.Fatalf("query scalar int: %v", err)
	}
	return count
}

func clauseSection(number int, heading, body string) string {
	return fmt.Sprintf("Section %d %s\n%s", number, heading, body)
}

func joinSections(sections ...string) string {
	return strings.Join(sections, "\n\n")
}

func lifecycleContractText() string {
	return joinSections(
		clauseSection(1, "Termination", "Either party may terminate for material breach after thirty days notice and a ten day cure period."),
		clauseSection(2, "Limitation of Liability", "The limitation of liability and aggregate liability cap shall not exceed fees paid in the preceding twelve months."),
		clauseSection(3, "Confidentiality", "Confidential information shall be used only for this agreement and protected from unauthorized disclosure."),
		clauseSection(4, "Governing Law", "This agreement is governed by the laws of the Kingdom of Saudi Arabia."),
		clauseSection(5, "Dispute Resolution", "Any dispute will be escalated to senior executives before mediation in Riyadh."),
		clauseSection(6, "Force Majeure", "A force majeure event beyond reasonable control suspends performance only while the event continues."),
		clauseSection(7, "Payment Terms", "Invoices are payable within thirty days after receipt of an undisputed invoice."),
		clauseSection(8, "Warranty", "Each party warrants that it has authority to enter the agreement and will perform its obligations professionally."),
		clauseSection(9, "Service Levels", "Service levels include uptime commitments, response times, and service credits for service failures."),
		clauseSection(10, "IP Ownership", "Work product created specifically for Clario is assigned to Clario upon payment."),
	)
}

func targetedAnalysisText() string {
	return joinSections(
		clauseSection(1, "Termination", "Either party may terminate for material breach after thirty days notice and a cure period."),
		clauseSection(2, "Limitation of Liability", "The limitation of liability establishes unlimited liability, no cap, excluding consequential damages, excluding indirect damages, and a waiver of liability."),
		clauseSection(3, "Confidentiality", "The parties shall protect confidential information and proprietary information from disclosure."),
		clauseSection(4, "Payment Terms", "Invoices are due on a net 90 basis and no penalty for late payment applies."),
		"The supplier may process personal data while providing the services.",
	)
}

func eightClauseText() string {
	return joinSections(
		clauseSection(1, "Termination", "Either party may terminate this agreement with thirty days notice after a material breach and a ten day cure period."),
		clauseSection(2, "Indemnification", "The supplier shall indemnify and hold harmless Clario against third-party claims caused by the supplier's negligence."),
		clauseSection(3, "Limitation of Liability", "The limitation of liability and aggregate liability cap shall not exceed fees paid during the prior contract year."),
		clauseSection(4, "Confidentiality", "Confidential information shall be used only for the agreement and protected from unauthorized disclosure."),
		clauseSection(5, "Force Majeure", "A force majeure event beyond reasonable control suspends performance until the event ends."),
		clauseSection(6, "Dispute Resolution", "Any dispute will be escalated to executives before mediation in Riyadh."),
		clauseSection(7, "Data Protection", "The supplier shall implement data protection safeguards for personal data and notify Clario of any data breach."),
		clauseSection(8, "Audit Rights", "Clario may audit and inspect relevant records once per year on reasonable notice."),
	)
}

func highRiskClauseText() string {
	return joinSections(
		clauseSection(1, "Termination", "Either party may terminate without cause, immediate, with no notice, at will, by unilateral election, and through automatic termination."),
		clauseSection(2, "Limitation of Liability", "The limitation of liability provides unlimited liability, no cap, no limitation, excluding consequential damages, excluding indirect damages, and a waiver of liability."),
		clauseSection(3, "Audit Rights", "There is no audit right and access to records is available only with consent and limited frequency."),
		clauseSection(4, "Confidentiality", "Confidential information shall remain protected and used only for this agreement."),
	)
}

func largeContractText() string {
	fillerSentence := "The parties will document responsibilities, operating procedures, remediation actions, acceptance criteria, reporting cadence, service governance, and implementation milestones in writing. "
	appendix := strings.Repeat(fillerSentence, 260)
	return joinSections(
		lifecycleContractText(),
		"Appendix A Background\n"+appendix,
	)
}
