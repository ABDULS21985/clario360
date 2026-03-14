package playbook

import (
	"sort"
	"time"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// Registry holds all built-in remediation playbooks and provides lookup by ID or finding type.
type Registry struct {
	byID          map[string]*model.Playbook
	byFindingType map[model.FindingType]*model.Playbook
	ordered       []*model.Playbook
}

// NewRegistry creates a Registry pre-loaded with all built-in playbooks.
func NewRegistry() *Registry {
	r := &Registry{
		byID:          make(map[string]*model.Playbook),
		byFindingType: make(map[model.FindingType]*model.Playbook),
	}
	for _, pb := range builtinPlaybooks() {
		cp := pb // make a distinct copy for the map
		r.byID[cp.ID] = &cp
		r.byFindingType[cp.FindingType] = &cp
		r.ordered = append(r.ordered, &cp)
	}
	return r
}

// Get returns a playbook by its unique identifier.
func (r *Registry) Get(id string) (*model.Playbook, bool) {
	pb, ok := r.byID[id]
	return pb, ok
}

// List returns all built-in playbooks sorted by ID.
func (r *Registry) List() []*model.Playbook {
	out := make([]*model.Playbook, len(r.ordered))
	copy(out, r.ordered)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// GetForFindingType returns the playbook mapped to a specific finding type.
func (r *Registry) GetForFindingType(findingType model.FindingType) (*model.Playbook, bool) {
	pb, ok := r.byFindingType[findingType]
	return pb, ok
}

// builtinPlaybooks returns the full set of 10 built-in remediation playbooks.
func builtinPlaybooks() []model.Playbook {
	return []model.Playbook{
		encryptSensitiveData(),
		revokeOverprivilegedAccess(),
		restrictNetworkExposure(),
		remediateShadowCopy(),
		enforcePIIControls(),
		handleClassificationDrift(),
		enforceDataRetention(),
		reduceBlastRadius(),
		postureGapGeneric(),
		staleAccessCleanup(),
	}
}

func encryptSensitiveData() model.Playbook {
	return model.Playbook{
		ID:               "encrypt-sensitive-data",
		Name:             "Encrypt Sensitive Data",
		Description:      "Applies encryption at rest and in transit to data assets missing required encryption controls, and establishes audit logging.",
		FindingType:      model.FindingEncryptionMissing,
		EstimatedMinutes: 30,
		RequiresApproval: false,
		AutoRollback:     false,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionNotifyOwner,
				Description:     "Notify the data asset owner about the missing encryption finding",
				Guidance:        "The owner will receive an email and in-app notification with finding details and the upcoming remediation plan.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Notification delivered to asset owner via at least one channel",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionCreateTicket,
				Description:     "Create a high-priority ITSM ticket for encryption remediation",
				Params:          map[string]any{"priority": "high"},
				Guidance:        "A ticket is created in the integrated ITSM system to track encryption remediation progress and ensure SLA compliance.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "ITSM ticket created with high priority and assigned to security-operations queue",
				FailureHandling: model.FailureHandlingRetry,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionEncryptAtRest,
				Description:     "Apply AES-256-GCM encryption at rest to the data asset",
				Params:          map[string]any{"algorithm": "AES-256-GCM"},
				Guidance:        "Encryption is applied via the platform KMS. All data columns are encrypted with a dedicated data encryption key.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Data asset encrypted at rest with AES-256-GCM using platform-managed KMS keys",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-4",
				Order:           4,
				Action:          model.StepActionEncryptInTransit,
				Description:     "Enforce TLS 1.3 for all in-transit data connections",
				Params:          map[string]any{"protocol": "TLS-1.3"},
				Guidance:        "All connections to this data asset will require TLS 1.3 minimum. Legacy plaintext endpoints are disabled.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "All data-in-transit paths enforcing TLS 1.3 with auto-provisioned certificates",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-5",
				Order:           5,
				Action:          model.StepActionEnableAuditLog,
				Description:     "Enable comprehensive audit logging to monitor encrypted data access",
				Params:          map[string]any{"retention_days": float64(90)},
				Guidance:        "Detailed audit logging captures all access events on the now-encrypted asset and forwards to the centralized SIEM.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Audit logging enabled with 90-day retention forwarding to centralized SIEM",
				FailureHandling: model.FailureHandlingSkip,
			},
		},
	}
}

func revokeOverprivilegedAccess() model.Playbook {
	return model.Playbook{
		ID:               "revoke-overprivileged-access",
		Name:             "Revoke Over-Privileged Access",
		Description:      "Removes excessive permissions from identities that have more access than required by their role, enforcing least-privilege.",
		FindingType:      model.FindingOverprivilegedAccess,
		EstimatedMinutes: 20,
		RequiresApproval: false,
		AutoRollback:     false,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionNotifyOwner,
				Description:     "Notify the asset owner about overprivileged access detected on their asset",
				Guidance:        "The owner is informed of the specific identities with excessive permissions and the planned remediation steps.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Owner notification delivered with list of overprivileged identities",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionScheduleReview,
				Description:     "Schedule a periodic access review to prevent privilege creep",
				Params:          map[string]any{"interval_days": float64(30)},
				Guidance:        "An access review is scheduled every 30 days. If a review is missed, permissions are auto-revoked as a safety measure.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "30-day recurring access review scheduled with auto-revoke on missed review",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionRevokeAccess,
				Description:     "Revoke all excess permissions that exceed the identity's required access",
				Params:          map[string]any{"scope": "excess_permissions"},
				Guidance:        "Only permissions exceeding the identity's role requirements are revoked. Core role permissions remain intact.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "All excess permissions revoked; active sessions using revoked permissions terminated",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-4",
				Order:           4,
				Action:          model.StepActionEnableAuditLog,
				Description:     "Enable audit logging on the asset to detect future privilege escalation",
				Params:          map[string]any{"retention_days": float64(90)},
				Guidance:        "Comprehensive logging is enabled to capture all permission changes and detect unauthorized escalation attempts.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Audit logging enabled capturing permission changes with 90-day retention",
				FailureHandling: model.FailureHandlingSkip,
			},
		},
	}
}

func restrictNetworkExposure() model.Playbook {
	return model.Playbook{
		ID:               "restrict-network-exposure",
		Name:             "Restrict Network Exposure",
		Description:      "Reduces the network attack surface for data assets with excessive exposure by applying firewall rules and network segmentation.",
		FindingType:      model.FindingExposureRisk,
		EstimatedMinutes: 15,
		RequiresApproval: false,
		AutoRollback:     false,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionCreateTicket,
				Description:     "Create a critical-priority ITSM ticket for exposure remediation",
				Params:          map[string]any{"priority": "critical"},
				Guidance:        "A critical-priority ticket is created immediately due to the severity of network exposure risks.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Critical-priority ITSM ticket created and assigned to network security team",
				FailureHandling: model.FailureHandlingRetry,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionRestrictNetwork,
				Description:     "Apply network restrictions to limit exposure to internal-only access",
				Params:          map[string]any{"target_exposure": "internal_only"},
				Guidance:        "Firewall rules are applied to block all public access paths. Only internal CIDR ranges are allowed.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Network exposure reduced to internal-only; all public endpoints blocked",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionNotifyOwner,
				Description:     "Notify the asset owner of the network restriction changes applied",
				Guidance:        "The owner is informed of the new network restrictions and provided instructions for requesting exceptions.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Owner notification delivered with details of network restrictions and exception process",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-4",
				Order:           4,
				Action:          model.StepActionEnableAuditLog,
				Description:     "Enable audit logging for network access attempts to the asset",
				Params:          map[string]any{"retention_days": float64(90)},
				Guidance:        "Network access audit logging captures all connection attempts, including blocked attempts from restricted sources.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Network access audit logging enabled with blocked-attempt capture",
				FailureHandling: model.FailureHandlingSkip,
			},
		},
	}
}

func remediateShadowCopy() model.Playbook {
	return model.Playbook{
		ID:               "remediate-shadow-copy",
		Name:             "Remediate Shadow Copy",
		Description:      "Quarantines unauthorized shadow copies of sensitive data and reclassifies them to enforce proper governance controls.",
		FindingType:      model.FindingShadowCopy,
		EstimatedMinutes: 25,
		RequiresApproval: true,
		AutoRollback:     true,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionNotifyOwner,
				Description:     "Notify the original data asset owner about the discovered shadow copy",
				Guidance:        "The owner of the source data asset is informed that an unauthorized copy has been detected and quarantine is imminent.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Source data owner notified of shadow copy discovery",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionQuarantine,
				Description:     "Quarantine the shadow copy to prevent further unauthorized access",
				Guidance:        "The shadow copy is isolated from all access paths except admin read-only. This prevents data exfiltration while investigation proceeds.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Shadow copy quarantined with all non-admin access paths blocked",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionCreateTicket,
				Description:     "Create an ITSM ticket for shadow copy investigation and disposition",
				Params:          map[string]any{"priority": "high"},
				Guidance:        "An investigation ticket is created to determine the origin, purpose, and proper disposition of the shadow copy.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Investigation ticket created with high priority for shadow copy disposition",
				FailureHandling: model.FailureHandlingRetry,
			},
			{
				ID:              "step-4",
				Order:           4,
				Action:          model.StepActionReclassify,
				Description:     "Reclassify the shadow copy to match the source data asset classification",
				Params:          map[string]any{"target_classification": "confidential"},
				Guidance:        "The shadow copy is reclassified to at least match the source asset classification level, triggering appropriate policy enforcement.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Shadow copy classification aligned with source data asset classification",
				FailureHandling: model.FailureHandlingAbort,
			},
		},
	}
}

func enforcePIIControls() model.Playbook {
	return model.Playbook{
		ID:               "enforce-pii-controls",
		Name:             "Enforce PII Controls",
		Description:      "Applies comprehensive PII protection controls including reclassification, encryption, audit logging, network restrictions, and periodic access reviews.",
		FindingType:      model.FindingPIIUnprotected,
		EstimatedMinutes: 40,
		RequiresApproval: false,
		AutoRollback:     false,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionReclassify,
				Description:     "Reclassify data asset containing PII to at least Confidential level",
				Params:          map[string]any{"target_classification": "confidential"},
				Guidance:        "Data assets containing PII must be classified at Confidential or higher to trigger appropriate policy controls.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Data asset reclassified to Confidential or higher with PII labels applied",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionEncryptAtRest,
				Description:     "Encrypt PII data at rest using AES-256-GCM",
				Params:          map[string]any{"algorithm": "AES-256-GCM"},
				Guidance:        "All columns containing PII are encrypted at rest. This is a regulatory requirement for most privacy frameworks.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "All PII columns encrypted at rest with AES-256-GCM",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionEnableAuditLog,
				Description:     "Enable detailed audit logging for all PII access events",
				Params:          map[string]any{"retention_days": float64(365)},
				Guidance:        "PII access logging requires extended retention (365 days) for regulatory compliance. All read/write/delete events are captured.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "PII access audit logging enabled with 365-day retention for regulatory compliance",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-4",
				Order:           4,
				Action:          model.StepActionRestrictNetwork,
				Description:     "Restrict network access to PII data to VPN-accessible only",
				Params:          map[string]any{"target_exposure": "vpn_accessible"},
				Guidance:        "PII data must not be accessible from the public internet. Access is restricted to VPN-connected clients only.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "PII data network access restricted to VPN-accessible endpoints only",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-5",
				Order:           5,
				Action:          model.StepActionScheduleReview,
				Description:     "Schedule quarterly access review for PII data assets",
				Params:          map[string]any{"interval_days": float64(90)},
				Guidance:        "Quarterly access reviews ensure ongoing compliance with PII access policies and detect privilege creep.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "90-day recurring access review scheduled for PII data asset",
				FailureHandling: model.FailureHandlingSkip,
			},
		},
	}
}

func handleClassificationDrift() model.Playbook {
	return model.Playbook{
		ID:               "handle-classification-drift",
		Name:             "Handle Classification Drift",
		Description:      "Corrects data assets whose effective classification has drifted from their assigned level due to content changes or policy updates.",
		FindingType:      model.FindingClassificationDrift,
		EstimatedMinutes: 20,
		RequiresApproval: false,
		AutoRollback:     false,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionReclassify,
				Description:     "Reclassify the data asset to its correct classification level based on current content analysis",
				Params:          map[string]any{"target_classification": "confidential"},
				Guidance:        "The classifier has detected content changes that warrant a higher classification. The asset is reclassified accordingly.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Data asset reclassified to correct level matching current content analysis",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionNotifyOwner,
				Description:     "Notify the asset owner about the classification change and its implications",
				Guidance:        "The owner is informed of the classification change, new governance controls that will be applied, and any access restrictions.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Owner notified of classification change with explanation of new governance controls",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionCreateTicket,
				Description:     "Create an ITSM ticket to track classification drift remediation",
				Params:          map[string]any{"priority": "medium"},
				Guidance:        "A tracking ticket is created to document the classification change and any follow-up actions required.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "ITSM ticket created documenting classification drift and remediation actions",
				FailureHandling: model.FailureHandlingRetry,
			},
			{
				ID:              "step-4",
				Order:           4,
				Action:          model.StepActionScheduleReview,
				Description:     "Schedule a follow-up review to verify classification accuracy after drift correction",
				Params:          map[string]any{"interval_days": float64(14)},
				Guidance:        "A 14-day follow-up review ensures the reclassification is stable and no further drift occurs.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "14-day follow-up review scheduled to verify classification stability",
				FailureHandling: model.FailureHandlingSkip,
			},
		},
	}
}

func enforceDataRetention() model.Playbook {
	return model.Playbook{
		ID:               "enforce-data-retention",
		Name:             "Enforce Data Retention",
		Description:      "Archives data that has exceeded its retention period, ensuring compliance with data lifecycle policies while maintaining audit trails.",
		FindingType:      model.FindingRetentionExpired,
		EstimatedMinutes: 35,
		RequiresApproval: true,
		AutoRollback:     false,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionNotifyOwner,
				Description:     "Notify the data owner that their asset has exceeded the retention period",
				Guidance:        "The owner is informed of the retention expiration and given a window to request an exception before archival proceeds.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Data owner notified of retention expiration with exception request instructions",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionArchiveData,
				Description:     "Archive the retention-expired data to cold storage with encryption",
				Params:          map[string]any{"archive_tier": "cold-storage"},
				Guidance:        "Data is moved to encrypted cold storage with a 24-hour retrieval SLA. A retention lock prevents premature deletion.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Data archived to encrypted cold storage with retention lock and 24-hour retrieval SLA",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionEnableAuditLog,
				Description:     "Enable audit logging on the archived data for compliance evidence",
				Params:          map[string]any{"retention_days": float64(730)},
				Guidance:        "Audit logs for archived data are retained for 2 years (730 days) to satisfy regulatory evidence requirements.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Audit logging enabled on archived data with 730-day retention for compliance evidence",
				FailureHandling: model.FailureHandlingSkip,
			},
		},
	}
}

func reduceBlastRadius() model.Playbook {
	return model.Playbook{
		ID:               "reduce-blast-radius",
		Name:             "Reduce Blast Radius",
		Description:      "Reduces the blast radius of a data asset by revoking excessive access, downgrading privileges, and establishing periodic reviews.",
		FindingType:      model.FindingBlastRadiusExcessive,
		EstimatedMinutes: 25,
		RequiresApproval: false,
		AutoRollback:     false,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionRevokeAccess,
				Description:     "Revoke access for identities outside the required access boundary",
				Params:          map[string]any{"scope": "out_of_boundary"},
				Guidance:        "Identities with no business justification for access are revoked immediately to reduce the potential impact of a breach.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Access revoked for all identities outside the required access boundary",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionDowngradeAccess,
				Description:     "Downgrade remaining access to the minimum required privilege level",
				Params:          map[string]any{"target_level": "read_only"},
				Guidance:        "Remaining identities are downgraded to read-only unless they have documented write requirements.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "All remaining access downgraded to read-only minimum privilege",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionScheduleReview,
				Description:     "Schedule monthly access reviews to prevent blast radius expansion",
				Params:          map[string]any{"interval_days": float64(30)},
				Guidance:        "Monthly reviews prevent the blast radius from expanding again through privilege creep.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "30-day recurring access review scheduled to monitor blast radius",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-4",
				Order:           4,
				Action:          model.StepActionEnableAuditLog,
				Description:     "Enable audit logging to track access patterns and detect re-expansion",
				Params:          map[string]any{"retention_days": float64(90)},
				Guidance:        "Audit logging enables detection of access pattern changes that might indicate blast radius re-expansion.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Access audit logging enabled with blast radius expansion alerting",
				FailureHandling: model.FailureHandlingSkip,
			},
		},
	}
}

func postureGapGeneric() model.Playbook {
	return model.Playbook{
		ID:               "posture-gap-generic",
		Name:             "Posture Gap - Generic Remediation",
		Description:      "Addresses generic data security posture gaps by creating tracking tickets, notifying stakeholders, and establishing ongoing monitoring.",
		FindingType:      model.FindingPostureGap,
		EstimatedMinutes: 15,
		RequiresApproval: false,
		AutoRollback:     false,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionCreateTicket,
				Description:     "Create an ITSM ticket to track the posture gap remediation",
				Params:          map[string]any{"priority": "medium"},
				Guidance:        "A tracking ticket is created with medium priority to ensure the posture gap is addressed within SLA.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "ITSM ticket created with medium priority and posture gap details",
				FailureHandling: model.FailureHandlingRetry,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionNotifyOwner,
				Description:     "Notify the asset owner about the identified posture gap",
				Guidance:        "The owner is informed of the specific posture gap, its risk implications, and recommended remediation actions.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Owner notification delivered with posture gap details and remediation guidance",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionEnableAuditLog,
				Description:     "Enable audit logging to monitor the asset for further posture degradation",
				Params:          map[string]any{"retention_days": float64(90)},
				Guidance:        "Audit logging provides visibility into ongoing posture changes and helps prevent further degradation.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Audit logging enabled to track posture-relevant events",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-4",
				Order:           4,
				Action:          model.StepActionScheduleReview,
				Description:     "Schedule a follow-up review to verify posture gap resolution",
				Params:          map[string]any{"interval_days": float64(14)},
				Guidance:        "A 14-day review is scheduled to verify the posture gap has been resolved and no new gaps have emerged.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "14-day follow-up review scheduled to verify posture gap resolution",
				FailureHandling: model.FailureHandlingSkip,
			},
		},
	}
}

func staleAccessCleanup() model.Playbook {
	return model.Playbook{
		ID:               "stale-access-cleanup",
		Name:             "Stale Access Cleanup",
		Description:      "Removes stale access grants that have not been used within the defined activity window, reducing the attack surface.",
		FindingType:      model.FindingStaleAccess,
		EstimatedMinutes: 20,
		RequiresApproval: false,
		AutoRollback:     false,
		Steps: []model.PlaybookStep{
			{
				ID:              "step-1",
				Order:           1,
				Action:          model.StepActionNotifyOwner,
				Description:     "Notify the asset owner about stale access grants identified on their asset",
				Guidance:        "The owner is informed which identities have not accessed the asset within the activity window and will have access revoked.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Owner notified with list of stale access grants and upcoming revocation timeline",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-2",
				Order:           2,
				Action:          model.StepActionScheduleReview,
				Description:     "Schedule a grace-period review before revoking stale access",
				Params:          map[string]any{"interval_days": float64(7)},
				Guidance:        "A 7-day grace period review allows affected users to re-confirm their access needs before revocation proceeds.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "7-day grace period review scheduled for stale access confirmation",
				FailureHandling: model.FailureHandlingSkip,
			},
			{
				ID:              "step-3",
				Order:           3,
				Action:          model.StepActionRevokeAccess,
				Description:     "Revoke all access grants that remain stale after the grace period",
				Params:          map[string]any{"scope": "stale_grants"},
				Guidance:        "Access grants not re-confirmed during the grace period are revoked. Users can request re-provisioning through standard access request workflows.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "All stale access grants revoked; affected users notified of re-provisioning process",
				FailureHandling: model.FailureHandlingAbort,
			},
			{
				ID:              "step-4",
				Order:           4,
				Action:          model.StepActionEnableAuditLog,
				Description:     "Enable audit logging to detect future access staleness",
				Params:          map[string]any{"retention_days": float64(90)},
				Guidance:        "Audit logging with access-frequency tracking detects newly stale access grants automatically.",
				Timeout:         5 * time.Minute,
				SuccessCriteria: "Audit logging enabled with access-frequency tracking for staleness detection",
				FailureHandling: model.FailureHandlingSkip,
			},
		},
	}
}
