package rca

import "strings"

// Recommender generates actionable recommendations based on root cause analysis.
type Recommender struct{}

// NewRecommender creates a new recommender.
func NewRecommender() *Recommender {
	return &Recommender{}
}

// ForSecurityAlert generates recommendations for a security alert RCA.
func (r *Recommender) ForSecurityAlert(rootCauseType string, chain []CausalStep) []Recommendation {
	switch rootCauseType {
	case "exposed_service":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Restrict network access to the exposed service.", Rationale: "The root cause was an exposed public-facing service that allowed initial access.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "immediate", Action: "Apply latest security patches to the affected service.", Rationale: "Exposed services are prime targets for known vulnerability exploits.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "short_term", Action: "Implement WAF rules to filter malicious traffic.", Rationale: "Web application firewall can block known attack patterns.", RootCauseType: rootCauseType},
			{Priority: 4, Category: "long_term", Action: "Review and reduce the attack surface by moving services behind VPN or internal networks.", Rationale: "Minimize internet-facing services to reduce exposure.", RootCauseType: rootCauseType},
		}
	case "credential_compromise":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Force password reset for all compromised accounts.", Rationale: "Compromised credentials were used for unauthorized access.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "immediate", Action: "Enable or enforce multi-factor authentication (MFA).", Rationale: "MFA prevents credential-only access even with leaked passwords.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "short_term", Action: "Review and revoke any sessions or tokens created during the compromise window.", Rationale: "Active sessions may still allow unauthorized access.", RootCauseType: rootCauseType},
			{Priority: 4, Category: "long_term", Action: "Implement credential monitoring and password breach detection.", Rationale: "Proactive detection of compromised credentials prevents future incidents.", RootCauseType: rootCauseType},
		}
	case "insider_threat":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Review and restrict the user's access permissions immediately.", Rationale: "The root cause indicates potential insider threat activity.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "immediate", Action: "Initiate HR investigation and document findings.", Rationale: "Insider threats require coordinated security and HR response.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "short_term", Action: "Enable enhanced monitoring and DLP controls for the affected user and similar roles.", Rationale: "Prevent further data exfiltration while investigation is ongoing.", RootCauseType: rootCauseType},
			{Priority: 4, Category: "long_term", Action: "Implement least-privilege access controls and regular access reviews.", Rationale: "Minimize the damage potential from insider threats.", RootCauseType: rootCauseType},
		}
	case "lateral_movement":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Isolate affected hosts and revoke compromised sessions.", Rationale: "Lateral movement detected — attacker is expanding their foothold.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "immediate", Action: "Reset credentials for all affected hosts and accounts.", Rationale: "Prevent further lateral movement with potentially harvested credentials.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "short_term", Action: "Implement network segmentation and microsegmentation.", Rationale: "Limit the blast radius of future lateral movement attempts.", RootCauseType: rootCauseType},
			{Priority: 4, Category: "long_term", Action: "Deploy endpoint detection and response (EDR) across all endpoints.", Rationale: "Real-time monitoring detects lateral movement techniques early.", RootCauseType: rootCauseType},
		}
	case "unpatched_vulnerability":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Apply the security patch or implement compensating controls.", Rationale: "The vulnerability was exploited to gain unauthorized access.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "immediate", Action: "Scan all similar assets for the same vulnerability.", Rationale: "The same vulnerability may exist on other systems.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "short_term", Action: "Review and improve the patch management process.", Rationale: "Timely patching would have prevented this exploitation.", RootCauseType: rootCauseType},
			{Priority: 4, Category: "long_term", Action: "Implement vulnerability scanning and automated patch deployment.", Rationale: "Continuous vulnerability management reduces exposure window.", RootCauseType: rootCauseType},
		}
	default:
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Contain the affected assets and isolate from the network.", Rationale: "Containment prevents further spread of the security incident.", RootCauseType: "unknown"},
			{Priority: 2, Category: "short_term", Action: "Conduct a thorough forensic investigation of affected systems.", Rationale: "Detailed investigation is needed to determine the full scope.", RootCauseType: "unknown"},
			{Priority: 3, Category: "long_term", Action: "Review and enhance security monitoring and detection capabilities.", Rationale: "Improved detection reduces time to identify future incidents.", RootCauseType: "unknown"},
		}
	}
}

// ForPipelineFailure generates recommendations for a pipeline failure RCA.
func (r *Recommender) ForPipelineFailure(rootCauseType string) []Recommendation {
	switch rootCauseType {
	case "upstream_failure":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Investigate and fix the upstream pipeline failure first.", Rationale: "This pipeline failed because its upstream dependency failed.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "short_term", Action: "Add dependency health checks before pipeline execution.", Rationale: "Pre-execution checks can fail fast instead of propagating errors.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "long_term", Action: "Implement circuit breakers for pipeline dependencies.", Rationale: "Circuit breakers prevent cascading failures across dependent pipelines.", RootCauseType: rootCauseType},
		}
	case "schema_drift":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Update pipeline schema mapping to match the new source schema.", Rationale: "The source schema changed, breaking the pipeline's extraction or loading logic.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "short_term", Action: "Add schema validation checks at pipeline start.", Rationale: "Early detection of schema changes prevents mid-execution failures.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "long_term", Action: "Implement schema change notifications from data sources.", Rationale: "Proactive schema monitoring allows ahead-of-time pipeline updates.", RootCauseType: rootCauseType},
		}
	case "connection_timeout":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Verify the data source is reachable and accepting connections.", Rationale: "The pipeline could not connect to its source.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "short_term", Action: "Increase connection timeout and add retry logic.", Rationale: "Transient network issues can be handled with retries.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "long_term", Action: "Set up source health monitoring with automated alerts.", Rationale: "Proactive monitoring detects source unavailability before pipeline runs.", RootCauseType: rootCauseType},
		}
	case "resource_exhaustion":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Reduce batch size or increase memory allocation for the pipeline.", Rationale: "The pipeline exhausted available resources processing data.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "short_term", Action: "Implement incremental processing instead of full loads.", Rationale: "Processing smaller data chunks reduces peak resource usage.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "long_term", Action: "Set up resource monitoring and auto-scaling for pipeline workloads.", Rationale: "Dynamic resource allocation handles variable data volumes.", RootCauseType: rootCauseType},
		}
	case "credential_expiry":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Rotate the expired credentials and update the data source configuration.", Rationale: "The pipeline's authentication credentials have expired.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "short_term", Action: "Set up credential expiration monitoring and alerts.", Rationale: "Early warning prevents pipeline failures due to expired credentials.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "long_term", Action: "Implement automated credential rotation.", Rationale: "Automated rotation eliminates human error in credential management.", RootCauseType: rootCauseType},
		}
	case "quality_gate":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Review the quality gate thresholds and failing data patterns.", Rationale: "The pipeline was blocked by a data quality gate.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "short_term", Action: "Add data profiling to identify source data anomalies early.", Rationale: "Profiling detects data issues before they reach quality gates.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "long_term", Action: "Implement automated data cleansing steps upstream of quality gates.", Rationale: "Automated cleansing reduces manual remediation effort.", RootCauseType: rootCauseType},
		}
	default:
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Review pipeline logs and error messages for specific failure details.", Rationale: "Detailed investigation needed to determine the exact root cause.", RootCauseType: "unknown"},
			{Priority: 2, Category: "short_term", Action: "Add comprehensive error handling and logging to the pipeline.", Rationale: "Better observability will speed future root cause investigations.", RootCauseType: "unknown"},
		}
	}
}

// ForQualityIssue generates recommendations for a data quality issue RCA.
func (r *Recommender) ForQualityIssue(rootCauseType string) []Recommendation {
	switch rootCauseType {
	case "upstream_quality":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Fix the upstream data quality issue that propagated downstream.", Rationale: "The quality failure originated from an upstream source.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "short_term", Action: "Add data quality checks at source ingestion points.", Rationale: "Catching quality issues at the source prevents downstream propagation.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "long_term", Action: "Implement data contracts with upstream data producers.", Rationale: "Formal data contracts define quality expectations and accountability.", RootCauseType: rootCauseType},
		}
	case "schema_change":
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Update quality rules to accommodate the new schema.", Rationale: "A source schema change caused the quality rules to fail.", RootCauseType: rootCauseType},
			{Priority: 2, Category: "short_term", Action: "Set up schema change detection and alerts.", Rationale: "Early notification of schema changes enables proactive rule updates.", RootCauseType: rootCauseType},
			{Priority: 3, Category: "long_term", Action: "Version quality rule sets alongside schema versions.", Rationale: "Coupled versioning ensures rules stay aligned with schema evolution.", RootCauseType: rootCauseType},
		}
	default:
		return []Recommendation{
			{Priority: 1, Category: "immediate", Action: "Review the failing quality rule and recent data changes.", Rationale: "Manual investigation needed to identify the data quality root cause.", RootCauseType: "unknown"},
			{Priority: 2, Category: "short_term", Action: "Add data profiling to track statistical drift in affected columns.", Rationale: "Statistical profiling detects gradual data degradation.", RootCauseType: "unknown"},
		}
	}
}

// ClassifySecurityRootCause determines the root cause type from the causal chain.
func ClassifySecurityRootCause(chain []CausalStep) string {
	if len(chain) == 0 {
		return "unknown"
	}

	// Find the root cause step (the one marked IsRootCause, or the first step)
	var rootStep *CausalStep
	for i := range chain {
		if chain[i].IsRootCause {
			rootStep = &chain[i]
			break
		}
	}
	if rootStep == nil {
		rootStep = &chain[0]
	}

	// Classify based on MITRE phase
	if rootStep.MITREPhase != "" {
		phase := strings.ToLower(rootStep.MITREPhase)
		switch {
		case phase == "reconnaissance" || phase == "resource-development" || phase == "initial-access":
			return "exposed_service"
		case phase == "credential-access":
			return "credential_compromise"
		case phase == "collection" || phase == "command-and-control" || phase == "exfiltration":
			return "insider_threat"
		case phase == "lateral-movement":
			return "lateral_movement"
		case phase == "execution" || phase == "privilege-escalation":
			return "unpatched_vulnerability"
		}
	}

	// Classify based on MITRE technique ID patterns
	if rootStep.MITRETechID != "" {
		techID := strings.ToLower(rootStep.MITRETechID)
		switch {
		case strings.HasPrefix(techID, "t1078"): // Valid Accounts
			return "credential_compromise"
		case strings.HasPrefix(techID, "t1190"): // Exploit Public-Facing Application
			return "exposed_service"
		case strings.HasPrefix(techID, "t1021") || strings.HasPrefix(techID, "t1570"): // Remote Services, Lateral Tool Transfer
			return "lateral_movement"
		}
	}

	// Classify based on event type and description keywords
	desc := strings.ToLower(rootStep.Description)
	switch {
	case strings.Contains(desc, "credential") || strings.Contains(desc, "password") || strings.Contains(desc, "brute"):
		return "credential_compromise"
	case strings.Contains(desc, "lateral") || strings.Contains(desc, "pivot"):
		return "lateral_movement"
	case strings.Contains(desc, "exploit") || strings.Contains(desc, "vulnerability"):
		return "unpatched_vulnerability"
	case strings.Contains(desc, "exfiltrat") || strings.Contains(desc, "insider"):
		return "insider_threat"
	}

	return "unknown"
}
