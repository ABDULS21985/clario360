package service

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

type builtinRuleTemplate struct {
	Slug              string
	Name              string
	Description       string
	RuleType          model.DetectionRuleType
	Severity          model.Severity
	RuleContent       json.RawMessage
	MITRETacticIDs    []string
	MITRETechniqueIDs []string
	Tags              []string
	BaseConfidence    float64
}

func (t builtinRuleTemplate) ToDetectionRule() *model.DetectionRule {
	templateID := t.Slug
	return &model.DetectionRule{
		ID:                uuid.New(),
		Name:              t.Name,
		Description:       t.Description,
		RuleType:          t.RuleType,
		Severity:          t.Severity,
		Enabled:           true,
		RuleContent:       t.RuleContent,
		MITRETacticIDs:    t.MITRETacticIDs,
		MITRETechniqueIDs: t.MITRETechniqueIDs,
		BaseConfidence:    t.BaseConfidence,
		Tags:              t.Tags,
		IsTemplate:        true,
		TemplateID:        &templateID,
	}
}

func builtinRuleTemplates() []builtinRuleTemplate {
	return []builtinRuleTemplate{
		{
			Slug:              "brute-force-ssh-login",
			Name:              "Brute Force SSH Login",
			Description:       "Detect repeated SSH authentication failures from the same source IP within five minutes.",
			RuleType:          model.RuleTypeThreshold,
			Severity:          model.SeverityHigh,
			RuleContent:       rawRule(`{"field":"source_ip","condition":{"type":"login_failed","dest_port":22,"source|in":["ids","ssh_log"]},"threshold":10,"window":"5m","metric":"count"}`),
			MITRETacticIDs:    []string{"TA0006"},
			MITRETechniqueIDs: []string{"T1110.001"},
			Tags:              []string{"authentication", "ssh", "brute_force"},
			BaseConfidence:    0.82,
		},
		{
			Slug:              "brute-force-rdp-login",
			Name:              "Brute Force RDP Login",
			Description:       "Detect repeated RDP login failures from the same source IP within five minutes.",
			RuleType:          model.RuleTypeThreshold,
			Severity:          model.SeverityHigh,
			RuleContent:       rawRule(`{"field":"source_ip","condition":{"type":"login_failed","dest_port":3389},"threshold":10,"window":"5m","metric":"count"}`),
			MITRETacticIDs:    []string{"TA0006", "TA0008"},
			MITRETechniqueIDs: []string{"T1110.001", "T1021.001"},
			Tags:              []string{"authentication", "rdp", "brute_force"},
			BaseConfidence:    0.82,
		},
		{
			Slug:              "suspicious-powershell-execution",
			Name:              "Suspicious PowerShell Execution",
			Description:       "Detect PowerShell commands using encoded, no-profile, or in-memory execution patterns while excluding service accounts.",
			RuleType:          model.RuleTypeSigma,
			Severity:          model.SeverityHigh,
			RuleContent:       rawRule(`{"detection":{"selection_proc":{"process|in":["powershell.exe","pwsh"]},"selection_encoded":{"command_line|contains":"-enc"},"selection_noprofile":{"command_line|contains":"-nop"},"selection_iex":{"command_line|contains":"IEX"},"selection_download":{"command_line|contains":"downloadstring"},"selection_invoke":{"command_line|contains":"invoke-expression"},"filter_service":{"user|in":["svc-backup","svc-monitor","svc-deploy"]},"condition":"selection_proc and (selection_encoded or selection_noprofile or selection_iex or selection_download or selection_invoke) and not filter_service"}}`),
			MITRETacticIDs:    []string{"TA0002"},
			MITRETechniqueIDs: []string{"T1059.001"},
			Tags:              []string{"powershell", "script", "execution"},
			BaseConfidence:    0.85,
		},
		{
			Slug:              "port-scanning-activity",
			Name:              "Port Scanning Activity",
			Description:       "Detect a source IP probing many distinct destination ports within one minute.",
			RuleType:          model.RuleTypeThreshold,
			Severity:          model.SeverityMedium,
			RuleContent:       rawRule(`{"field":"source_ip","condition":{"source|in":["firewall","ids"]},"threshold":20,"window":"1m","metric":"distinct(dest_port)"}`),
			MITRETacticIDs:    []string{"TA0007"},
			MITRETechniqueIDs: []string{"T1046"},
			Tags:              []string{"network", "scan", "discovery"},
			BaseConfidence:    0.78,
		},
		{
			Slug:              "data-exfiltration-large-outbound-transfer",
			Name:              "Data Exfiltration — Large Outbound Transfer",
			Description:       "Detect abnormal outbound transfer volume for an asset based on its one-hour baseline.",
			RuleType:          model.RuleTypeAnomaly,
			Severity:          model.SeverityCritical,
			RuleContent:       rawRule(`{"metric":"bytes_transferred","group_by":"asset_id","window":"1h","z_score_threshold":3.0,"min_baseline_samples":100,"direction":"above"}`),
			MITRETacticIDs:    []string{"TA0010"},
			MITRETechniqueIDs: []string{"T1048"},
			Tags:              []string{"exfiltration", "network", "anomaly"},
			BaseConfidence:    0.88,
		},
		{
			Slug:              "known-malicious-ip-connection",
			Name:              "Known Malicious IP Connection",
			Description:       "Indicator-based alert generated when source or destination IP matches an active malicious indicator.",
			RuleType:          model.RuleTypeSigma,
			Severity:          model.SeverityCritical,
			RuleContent:       rawRule(`{"detection":{"selection":{"source_ip|exists":true},"condition":"selection"}}`),
			MITRETacticIDs:    []string{"TA0011"},
			MITRETechniqueIDs: []string{"T1071"},
			Tags:              []string{"indicator_matcher", "ioc", "network"},
			BaseConfidence:    0.92,
		},
		{
			Slug:              "unusual-login-time",
			Name:              "Unusual Login Time",
			Description:       "Detect logins at hours that deviate significantly from the user's historical pattern.",
			RuleType:          model.RuleTypeAnomaly,
			Severity:          model.SeverityMedium,
			RuleContent:       rawRule(`{"metric":"login_hour","group_by":"username","window":"24h","z_score_threshold":2.0,"min_baseline_samples":30,"direction":"both"}`),
			MITRETacticIDs:    []string{"TA0001", "TA0003"},
			MITRETechniqueIDs: []string{"T1078"},
			Tags:              []string{"authentication", "anomaly", "identity"},
			BaseConfidence:    0.75,
		},
		{
			Slug:              "privilege-escalation-attempt",
			Name:              "Privilege Escalation Attempt",
			Description:       "Detect suspicious privilege escalation commands and binaries, excluding known admin-group executions.",
			RuleType:          model.RuleTypeSigma,
			Severity:          model.SeverityCritical,
			RuleContent:       rawRule(`{"detection":{"selection_sudo":{"command_line|contains":"sudo","command_line|re":"(?i)(/bin/bash|/bin/sh|su\\s+-)"},"selection_pkexec":{"process|in":["pkexec","doas"]},"filter_admin":{"raw.user_in_admin_group":true},"condition":"(selection_sudo or selection_pkexec) and not filter_admin"}}`),
			MITRETacticIDs:    []string{"TA0004"},
			MITRETechniqueIDs: []string{"T1068", "T1548"},
			Tags:              []string{"privilege_escalation", "linux", "execution"},
			BaseConfidence:    0.90,
		},
		{
			Slug:              "dns-tunneling-detection",
			Name:              "DNS Tunneling Detection",
			Description:       "Detect anomalous DNS query volume indicating potential tunneling or covert channels.",
			RuleType:          model.RuleTypeAnomaly,
			Severity:          model.SeverityHigh,
			RuleContent:       rawRule(`{"metric":"dns_query_count","group_by":"asset_id","window":"1h","z_score_threshold":3.0,"min_baseline_samples":100,"direction":"above"}`),
			MITRETacticIDs:    []string{"TA0011"},
			MITRETechniqueIDs: []string{"T1071.004"},
			Tags:              []string{"dns", "tunneling", "c2"},
			BaseConfidence:    0.84,
		},
		{
			Slug:              "lateral-movement-via-smb",
			Name:              "Lateral Movement via SMB",
			Description:       "Detect SMB connections from one source to many distinct destination systems in ten minutes.",
			RuleType:          model.RuleTypeThreshold,
			Severity:          model.SeverityHigh,
			RuleContent:       rawRule(`{"field":"source_ip","condition":{"dest_port":445,"protocol":"tcp"},"threshold":5,"window":"10m","metric":"distinct(dest_ip)"}`),
			MITRETacticIDs:    []string{"TA0008"},
			MITRETechniqueIDs: []string{"T1021.002"},
			Tags:              []string{"smb", "lateral_movement", "network"},
			BaseConfidence:    0.83,
		},
		{
			Slug:              "ransomware-file-activity",
			Name:              "Ransomware File Activity",
			Description:       "Detect burst file renames to common ransomware extensions on the same asset within five minutes.",
			RuleType:          model.RuleTypeSigma,
			Severity:          model.SeverityCritical,
			RuleContent:       rawRule(`{"detection":{"selection":{"type":"file_rename","file_path|re":"(?i)\\.(encrypted|locked|crypto|zzzzz|locky|cerber)$"},"condition":"selection"},"timeframe":"5m","threshold":100}`),
			MITRETacticIDs:    []string{"TA0040"},
			MITRETechniqueIDs: []string{"T1486"},
			Tags:              []string{"ransomware", "filesystem", "impact"},
			BaseConfidence:    0.94,
		},
		{
			Slug:              "web-shell-detection",
			Name:              "Web Shell Detection",
			Description:       "Detect web-server parent processes spawning interactive shells or script interpreters.",
			RuleType:          model.RuleTypeSigma,
			Severity:          model.SeverityCritical,
			RuleContent:       rawRule(`{"detection":{"selection_proc":{"process|in":["cmd.exe","powershell.exe","bash","sh"]},"selection_parent":{"parent_process|in":["w3wp.exe","httpd","apache2","nginx","tomcat"]},"condition":"selection_proc and selection_parent"}}`),
			MITRETacticIDs:    []string{"TA0003"},
			MITRETechniqueIDs: []string{"T1505.003"},
			Tags:              []string{"web_shell", "server", "execution"},
			BaseConfidence:    0.91,
		},
		{
			Slug:              "credential-dumping",
			Name:              "Credential Dumping",
			Description:       "Detect common credential dumping tools and command-line patterns.",
			RuleType:          model.RuleTypeSigma,
			Severity:          model.SeverityCritical,
			RuleContent:       rawRule(`{"detection":{"selection_proc":{"process|in":["mimikatz","procdump","lsass"]},"selection_cmd":{"command_line|contains":"sekurlsa"},"selection_dump":{"command_line|contains":"lsadump"},"selection_sam":{"command_line|contains":"SAM"},"condition":"selection_proc or selection_cmd or selection_dump or selection_sam"}}`),
			MITRETacticIDs:    []string{"TA0006"},
			MITRETechniqueIDs: []string{"T1003"},
			Tags:              []string{"credentials", "dumping", "windows"},
			BaseConfidence:    0.93,
		},
		{
			Slug:              "c2-beaconing-pattern",
			Name:              "C2 Beaconing Pattern",
			Description:       "Detect unusually regular connection intervals suggestive of beaconing to a remote host.",
			RuleType:          model.RuleTypeAnomaly,
			Severity:          model.SeverityHigh,
			RuleContent:       rawRule(`{"metric":"connection_interval_regularity","group_by":"source_ip","window":"1h","z_score_threshold":2.5,"min_baseline_samples":50,"direction":"below"}`),
			MITRETacticIDs:    []string{"TA0011"},
			MITRETechniqueIDs: []string{"T1571", "T1071"},
			Tags:              []string{"c2", "beaconing", "network"},
			BaseConfidence:    0.86,
		},
		{
			Slug:              "unauthorized-cloud-api-call",
			Name:              "Unauthorized Cloud API Call",
			Description:       "Detect privileged cloud API calls by accounts outside the approved cloud administrator list.",
			RuleType:          model.RuleTypeSigma,
			Severity:          model.SeverityHigh,
			RuleContent:       rawRule(`{"detection":{"selection_source":{"source":"cloud_trail"},"selection_action":{"type|in":["CreateUser","AttachPolicy","AssumeRole","CreateAccessKey"]},"filter_admin":{"raw.user|in":["cloud-admin","infra-admin","security-admin"]},"condition":"selection_source and selection_action and not filter_admin"}}`),
			MITRETacticIDs:    []string{"TA0001", "TA0003", "TA0004"},
			MITRETechniqueIDs: []string{"T1078.004"},
			Tags:              []string{"cloud", "iam", "api"},
			BaseConfidence:    0.84,
		},
	}
}

func rawRule(value string) json.RawMessage {
	return json.RawMessage(value)
}

// BuiltinTenantRuleSeeds returns the built-in detection rule catalog as
// tenant-scoped defaults suitable for idempotent provisioning.
func BuiltinTenantRuleSeeds() []*model.DetectionRule {
	templates := builtinRuleTemplates()
	out := make([]*model.DetectionRule, 0, len(templates))
	for _, template := range templates {
		rule := template.ToDetectionRule()
		rule.IsTemplate = false
		rule.TemplateID = nil
		rule.TenantID = nil
		out = append(out, rule)
	}
	return out
}
