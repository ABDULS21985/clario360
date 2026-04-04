package mitre

import (
	"strings"
	"time"
)

// Framework metadata — update these when refreshing the embedded catalog.
const (
	// FrameworkVersion is the ATT&CK version this catalog is based on.
	FrameworkVersion = "15.1"
	// FrameworkUpdatedAt is the date the embedded catalog was last refreshed (ISO 8601).
	FrameworkUpdatedAt = "2024-10-31"
)

// frameworkUpdatedTime is the parsed form of FrameworkUpdatedAt.
var frameworkUpdatedTime, _ = time.Parse("2006-01-02", FrameworkUpdatedAt)

// FrameworkMeta returns version metadata about the embedded catalog.
func FrameworkMeta() FrameworkMetadata {
	staleDays := int(time.Since(frameworkUpdatedTime).Hours() / 24)
	return FrameworkMetadata{
		Version:        FrameworkVersion,
		UpdatedAt:      FrameworkUpdatedAt,
		TacticCount:    len(tactics),
		TechniqueCount: len(techniques),
		StaleDays:      staleDays,
		IsStale:        staleDays > 180,
	}
}

// FrameworkMetadata holds version and freshness info for the embedded ATT&CK catalog.
type FrameworkMetadata struct {
	Version        string `json:"version"`
	UpdatedAt      string `json:"updated_at"`
	TacticCount    int    `json:"tactic_count"`
	TechniqueCount int    `json:"technique_count"`
	StaleDays      int    `json:"stale_days"`
	IsStale        bool   `json:"is_stale"`
}

// Tactic represents a MITRE ATT&CK tactic.
type Tactic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"short_name"`
	Description string `json:"description"`
}

// Technique represents a MITRE ATT&CK technique or sub-technique.
type Technique struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TacticIDs   []string `json:"tactic_ids"`
	Platforms   []string `json:"platforms"`
	DataSources []string `json:"data_sources"`
}

var tactics = []Tactic{
	{ID: "TA0043", Name: "Reconnaissance", ShortName: "reconnaissance", Description: "Adversaries gather information they can use to plan future operations."},
	{ID: "TA0042", Name: "Resource Development", ShortName: "resource-development", Description: "Adversaries establish resources they need to support operations."},
	{ID: "TA0001", Name: "Initial Access", ShortName: "initial-access", Description: "Adversaries try to get into the target environment."},
	{ID: "TA0002", Name: "Execution", ShortName: "execution", Description: "Adversaries run malicious code in a target environment."},
	{ID: "TA0003", Name: "Persistence", ShortName: "persistence", Description: "Adversaries maintain long-term access to systems."},
	{ID: "TA0004", Name: "Privilege Escalation", ShortName: "privilege-escalation", Description: "Adversaries obtain higher-level permissions."},
	{ID: "TA0005", Name: "Defense Evasion", ShortName: "defense-evasion", Description: "Adversaries avoid detection or bypass defenses."},
	{ID: "TA0006", Name: "Credential Access", ShortName: "credential-access", Description: "Adversaries steal or abuse credentials."},
	{ID: "TA0007", Name: "Discovery", ShortName: "discovery", Description: "Adversaries learn about the environment they compromised."},
	{ID: "TA0008", Name: "Lateral Movement", ShortName: "lateral-movement", Description: "Adversaries move between systems in an environment."},
	{ID: "TA0009", Name: "Collection", ShortName: "collection", Description: "Adversaries gather data of interest to their objective."},
	{ID: "TA0011", Name: "Command and Control", ShortName: "command-and-control", Description: "Adversaries communicate with compromised systems."},
	{ID: "TA0010", Name: "Exfiltration", ShortName: "exfiltration", Description: "Adversaries steal data from the environment."},
	{ID: "TA0040", Name: "Impact", ShortName: "impact", Description: "Adversaries manipulate, interrupt, or destroy systems and data."},
}

var techniques = []Technique{
	{ID: "T1595", Name: "Active Scanning", Description: "Scan victim infrastructure for exposed services.", TacticIDs: []string{"TA0043"}, Platforms: []string{"Network"}, DataSources: []string{"Network Traffic"}},
	{ID: "T1592", Name: "Gather Victim Host Information", Description: "Collect host-specific details before intrusion.", TacticIDs: []string{"TA0043"}, Platforms: []string{"PRE"}, DataSources: []string{"Internet Scan"}},
	{ID: "T1583", Name: "Acquire Infrastructure", Description: "Obtain infrastructure to support operations.", TacticIDs: []string{"TA0042"}, Platforms: []string{"PRE"}, DataSources: []string{"DNS Records"}},
	{ID: "T1587", Name: "Develop Capabilities", Description: "Create malware, exploits, or payloads to support operations.", TacticIDs: []string{"TA0042"}, Platforms: []string{"PRE"}, DataSources: []string{"Threat Intelligence"}},
	{ID: "T1190", Name: "Exploit Public-Facing Application", Description: "Exploit an internet-facing application to gain access.", TacticIDs: []string{"TA0001"}, Platforms: []string{"Linux", "Windows", "Cloud"}, DataSources: []string{"Web Logs", "Network Traffic"}},
	{ID: "T1566", Name: "Phishing", Description: "Send fraudulent content to trick users into providing access.", TacticIDs: []string{"TA0001"}, Platforms: []string{"Office 365", "Windows", "SaaS"}, DataSources: []string{"Email Gateway", "Mailbox Activity"}},
	{ID: "T1078", Name: "Valid Accounts", Description: "Use legitimate credentials for access.", TacticIDs: []string{"TA0001", "TA0003", "TA0004", "TA0011"}, Platforms: []string{"Windows", "Linux", "Cloud"}, DataSources: []string{"Authentication Logs"}},
	{ID: "T1133", Name: "External Remote Services", Description: "Gain access through exposed VPN, RDP, or other remote services.", TacticIDs: []string{"TA0001"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"VPN Logs", "RDP Logs"}},
	{ID: "T1059", Name: "Command and Scripting Interpreter", Description: "Use scripting or shell interpreters to run code.", TacticIDs: []string{"TA0002"}, Platforms: []string{"Windows", "Linux", "macOS"}, DataSources: []string{"Process Creation"}},
	{ID: "T1059.001", Name: "PowerShell", Description: "Run commands and scripts through PowerShell.", TacticIDs: []string{"TA0002"}, Platforms: []string{"Windows"}, DataSources: []string{"PowerShell Logs", "Process Creation"}},
	{ID: "T1053", Name: "Scheduled Task/Job", Description: "Use task schedulers or cron to execute code.", TacticIDs: []string{"TA0002", "TA0003", "TA0004"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Task Scheduler", "Cron"}},
	{ID: "T1204", Name: "User Execution", Description: "Rely on a user to execute malicious content.", TacticIDs: []string{"TA0002"}, Platforms: []string{"Windows", "macOS"}, DataSources: []string{"Email", "Process Creation"}},
	{ID: "T1547", Name: "Boot or Logon Autostart Execution", Description: "Run code automatically at boot or logon.", TacticIDs: []string{"TA0003", "TA0005"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Registry", "Service Creation"}},
	{ID: "T1505.003", Name: "Web Shell", Description: "Deploy a malicious script to a web server for persistence.", TacticIDs: []string{"TA0003"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Web Logs", "Process Creation"}},
	{ID: "T1136", Name: "Create Account", Description: "Create accounts for persistent access.", TacticIDs: []string{"TA0003"}, Platforms: []string{"Windows", "Linux", "Cloud"}, DataSources: []string{"Identity Logs"}},
	{ID: "T1098", Name: "Account Manipulation", Description: "Modify account settings or memberships.", TacticIDs: []string{"TA0003", "TA0004"}, Platforms: []string{"Windows", "Linux", "Cloud"}, DataSources: []string{"Identity Logs", "Directory Service"}},
	{ID: "T1068", Name: "Exploitation for Privilege Escalation", Description: "Exploit a vulnerability to gain higher privileges.", TacticIDs: []string{"TA0004"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Kernel Logs", "Process Creation"}},
	{ID: "T1548", Name: "Abuse Elevation Control Mechanism", Description: "Bypass or misuse elevation controls such as UAC or sudo.", TacticIDs: []string{"TA0004"}, Platforms: []string{"Windows", "Linux", "macOS"}, DataSources: []string{"Process Creation"}},
	{ID: "T1611", Name: "Escape to Host", Description: "Escape from a container or virtualized boundary to the host.", TacticIDs: []string{"TA0004"}, Platforms: []string{"Containers"}, DataSources: []string{"Container Logs", "Kernel Logs"}},
	{ID: "T1484", Name: "Domain or Tenant Policy Modification", Description: "Modify domain or cloud policies to gain privileges.", TacticIDs: []string{"TA0004", "TA0005"}, Platforms: []string{"Windows", "Azure AD"}, DataSources: []string{"Policy Changes"}},
	{ID: "T1027", Name: "Obfuscated Files or Information", Description: "Hide payloads or commands through encoding or obfuscation.", TacticIDs: []string{"TA0005"}, Platforms: []string{"Windows", "Linux", "macOS"}, DataSources: []string{"File Metadata", "Process Creation"}},
	{ID: "T1070", Name: "Indicator Removal on Host", Description: "Delete logs or artifacts to conceal activity.", TacticIDs: []string{"TA0005"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"File Deletion", "Audit Logs"}},
	{ID: "T1036", Name: "Masquerading", Description: "Rename or disguise files and processes to appear legitimate.", TacticIDs: []string{"TA0005"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Process Creation", "File Metadata"}},
	{ID: "T1562", Name: "Impair Defenses", Description: "Disable or tamper with security tooling.", TacticIDs: []string{"TA0005"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Service Control", "AV Logs"}},
	{ID: "T1003", Name: "OS Credential Dumping", Description: "Dump credentials from OS stores such as LSASS or SAM.", TacticIDs: []string{"TA0006"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Process Creation", "Memory Access"}},
	{ID: "T1110", Name: "Brute Force", Description: "Attempt multiple passwords or authentication guesses.", TacticIDs: []string{"TA0006"}, Platforms: []string{"Windows", "Linux", "Cloud"}, DataSources: []string{"Authentication Logs"}},
	{ID: "T1110.001", Name: "Password Guessing", Description: "Guess passwords directly against an authentication service.", TacticIDs: []string{"TA0006"}, Platforms: []string{"Windows", "Linux", "Cloud"}, DataSources: []string{"Authentication Logs"}},
	{ID: "T1555", Name: "Credentials from Password Stores", Description: "Extract credentials from password managers or browsers.", TacticIDs: []string{"TA0006"}, Platforms: []string{"Windows", "macOS"}, DataSources: []string{"File Access"}},
	{ID: "T1046", Name: "Network Service Discovery", Description: "Scan for available services on remote hosts.", TacticIDs: []string{"TA0007"}, Platforms: []string{"Windows", "Linux", "Network"}, DataSources: []string{"Network Traffic"}},
	{ID: "T1087", Name: "Account Discovery", Description: "Enumerate local, domain, or cloud accounts.", TacticIDs: []string{"TA0007"}, Platforms: []string{"Windows", "Linux", "Cloud"}, DataSources: []string{"Directory Service", "Command History"}},
	{ID: "T1018", Name: "Remote System Discovery", Description: "Identify remote systems and network topology.", TacticIDs: []string{"TA0007"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Network Traffic", "Command History"}},
	{ID: "T1518", Name: "Software Discovery", Description: "Enumerate installed software and security tools.", TacticIDs: []string{"TA0007"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Process Creation", "Registry"}},
	{ID: "T1021", Name: "Remote Services", Description: "Use remote services to move laterally.", TacticIDs: []string{"TA0008"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Authentication Logs", "Network Traffic"}},
	{ID: "T1021.001", Name: "Remote Desktop Protocol", Description: "Use RDP to access remote Windows systems.", TacticIDs: []string{"TA0008"}, Platforms: []string{"Windows"}, DataSources: []string{"RDP Logs"}},
	{ID: "T1021.002", Name: "SMB/Windows Admin Shares", Description: "Use SMB or admin shares for lateral movement.", TacticIDs: []string{"TA0008"}, Platforms: []string{"Windows"}, DataSources: []string{"SMB Logs", "Network Traffic"}},
	{ID: "T1550", Name: "Use Alternate Authentication Material", Description: "Authenticate with tokens, hashes, or tickets.", TacticIDs: []string{"TA0008"}, Platforms: []string{"Windows", "Cloud"}, DataSources: []string{"Authentication Logs"}},
	{ID: "T1005", Name: "Data from Local System", Description: "Collect files or data from the compromised host.", TacticIDs: []string{"TA0009"}, Platforms: []string{"Windows", "Linux", "macOS"}, DataSources: []string{"File Access"}},
	{ID: "T1114", Name: "Email Collection", Description: "Collect email from local or remote mail stores.", TacticIDs: []string{"TA0009"}, Platforms: []string{"Office 365", "Exchange"}, DataSources: []string{"Mailbox Activity"}},
	{ID: "T1119", Name: "Automated Collection", Description: "Use scripted or automated collection of data.", TacticIDs: []string{"TA0009"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Process Creation", "File Access"}},
	{ID: "T1039", Name: "Data from Network Shared Drive", Description: "Access data from SMB or other shared drives.", TacticIDs: []string{"TA0009"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"File Access", "SMB Logs"}},
	{ID: "T1071", Name: "Application Layer Protocol", Description: "Communicate with C2 over common application protocols.", TacticIDs: []string{"TA0011"}, Platforms: []string{"Windows", "Linux", "Cloud"}, DataSources: []string{"DNS Logs", "Proxy Logs", "Network Traffic"}},
	{ID: "T1071.001", Name: "Web Protocols", Description: "Use HTTP or HTTPS for C2.", TacticIDs: []string{"TA0011"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Proxy Logs", "Web Logs"}},
	{ID: "T1071.004", Name: "DNS", Description: "Use DNS for command and control or tunneling.", TacticIDs: []string{"TA0011"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"DNS Logs"}},
	{ID: "T1571", Name: "Non-Standard Port", Description: "Use uncommon ports for C2 traffic.", TacticIDs: []string{"TA0011"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Network Traffic"}},
	{ID: "T1048", Name: "Exfiltration Over Alternative Protocol", Description: "Exfiltrate data using a non-primary protocol.", TacticIDs: []string{"TA0010"}, Platforms: []string{"Windows", "Linux", "Cloud"}, DataSources: []string{"Network Traffic"}},
	{ID: "T1041", Name: "Exfiltration Over C2 Channel", Description: "Steal data over an established command channel.", TacticIDs: []string{"TA0010"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Network Traffic", "Proxy Logs"}},
	{ID: "T1020", Name: "Automated Exfiltration", Description: "Automatically send collected data outside the network.", TacticIDs: []string{"TA0010"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Process Creation", "Network Traffic"}},
	{ID: "T1537", Name: "Transfer Data to Cloud Account", Description: "Move data into attacker-controlled cloud storage.", TacticIDs: []string{"TA0010"}, Platforms: []string{"Cloud", "Windows", "Linux"}, DataSources: []string{"Cloud Logs", "Proxy Logs"}},
	{ID: "T1486", Name: "Data Encrypted for Impact", Description: "Encrypt data to disrupt availability.", TacticIDs: []string{"TA0040"}, Platforms: []string{"Windows", "Linux", "macOS"}, DataSources: []string{"File Rename", "Process Creation"}},
	{ID: "T1490", Name: "Inhibit System Recovery", Description: "Delete backups or recovery mechanisms.", TacticIDs: []string{"TA0040"}, Platforms: []string{"Windows", "Linux"}, DataSources: []string{"Process Creation", "Backup Logs"}},
	{ID: "T1498", Name: "Network Denial of Service", Description: "Disrupt network availability through traffic volume or protocol abuse.", TacticIDs: []string{"TA0040"}, Platforms: []string{"Network"}, DataSources: []string{"Network Traffic"}},
	{ID: "T1565", Name: "Data Manipulation", Description: "Alter or corrupt data to affect operations or integrity.", TacticIDs: []string{"TA0040"}, Platforms: []string{"Windows", "Linux", "Cloud"}, DataSources: []string{"Database Logs", "File Access"}},
	{ID: "T1078.004", Name: "Cloud Accounts", Description: "Use valid cloud identities for access or persistence.", TacticIDs: []string{"TA0001", "TA0003", "TA0004", "TA0011"}, Platforms: []string{"Cloud"}, DataSources: []string{"Cloud Audit Logs"}},
	{ID: "T1531", Name: "Account Access Removal", Description: "Lock out or disable accounts to disrupt operations.", TacticIDs: []string{"TA0040"}, Platforms: []string{"Windows", "Cloud"}, DataSources: []string{"Identity Logs"}},
}

// AllTactics returns the embedded ATT&CK tactic catalog.
func AllTactics() []Tactic {
	out := make([]Tactic, len(tactics))
	copy(out, tactics)
	return out
}

// AllTechniques returns the embedded ATT&CK technique catalog.
func AllTechniques() []Technique {
	out := make([]Technique, len(techniques))
	copy(out, techniques)
	return out
}

// TechniquesByTactic filters techniques for a single tactic.
func TechniquesByTactic(tacticID string) []Technique {
	return TechniquesByTactics([]string{tacticID})
}

// TechniquesByTactics filters techniques matching any of the given tactic IDs.
func TechniquesByTactics(tacticIDs []string) []Technique {
	set := make(map[string]struct{}, len(tacticIDs))
	for _, id := range tacticIDs {
		set[strings.ToLower(id)] = struct{}{}
	}
	results := make([]Technique, 0)
	for _, technique := range techniques {
		for _, candidate := range technique.TacticIDs {
			if _, ok := set[strings.ToLower(candidate)]; ok {
				results = append(results, technique)
				break
			}
		}
	}
	return results
}

// TechniqueByID returns the technique matching the given ATT&CK ID.
func TechniqueByID(id string) (*Technique, bool) {
	for _, technique := range techniques {
		if strings.EqualFold(technique.ID, id) {
			copyValue := technique
			return &copyValue, true
		}
	}
	return nil, false
}

// TacticByID returns the tactic matching the given ATT&CK ID.
func TacticByID(id string) (*Tactic, bool) {
	for _, tactic := range tactics {
		if strings.EqualFold(tactic.ID, id) {
			copyValue := tactic
			return &copyValue, true
		}
	}
	return nil, false
}
