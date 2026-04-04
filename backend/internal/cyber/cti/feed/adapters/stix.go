package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// STIXAdapter parses a STIX 2.1 bundle and extracts indicators.
type STIXAdapter struct{}

func NewSTIXAdapter() *STIXAdapter { return &STIXAdapter{} }

func (a *STIXAdapter) SourceType() string { return "stix" }

// STIX 2.1 structures (subset)
type stixBundle struct {
	Type    string           `json:"type"`
	ID      string           `json:"id"`
	Objects []stixObject     `json:"objects"`
}

type stixObject struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Pattern     string    `json:"pattern,omitempty"`
	PatternType string    `json:"pattern_type,omitempty"`
	Created     time.Time `json:"created,omitempty"`
	Modified    time.Time `json:"modified,omitempty"`
	ValidFrom   time.Time `json:"valid_from,omitempty"`
	Confidence  int       `json:"confidence,omitempty"`
	Labels      []string  `json:"labels,omitempty"`
	// Kill chain
	KillChainPhases []struct {
		KillChainName string `json:"kill_chain_name"`
		PhaseName     string `json:"phase_name"`
	} `json:"kill_chain_phases,omitempty"`
	// External references
	ExternalReferences []struct {
		SourceName  string `json:"source_name"`
		ExternalID  string `json:"external_id"`
		URL         string `json:"url,omitempty"`
	} `json:"external_references,omitempty"`
}

var (
	ipv4Pattern   = regexp.MustCompile(`\[ipv4-addr:value\s*=\s*'([^']+)'\]`)
	domainPattern = regexp.MustCompile(`\[domain-name:value\s*=\s*'([^']+)'\]`)
	urlPattern    = regexp.MustCompile(`\[url:value\s*=\s*'([^']+)'\]`)
	hashPattern   = regexp.MustCompile(`\[file:hashes\.'([^']+)'\s*=\s*'([^']+)'\]`)
	emailPattern  = regexp.MustCompile(`\[email-addr:value\s*=\s*'([^']+)'\]`)
)

func (a *STIXAdapter) Parse(_ context.Context, raw []byte) ([]NormalizedIndicator, error) {
	var bundle stixBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return nil, fmt.Errorf("stix adapter: unmarshal bundle: %w", err)
	}

	if bundle.Type != "bundle" {
		return nil, fmt.Errorf("stix adapter: expected type 'bundle', got '%s'", bundle.Type)
	}

	// Build lookup maps for attack-patterns and malware by ID
	mitreMap := make(map[string]string) // stix-id → technique-id
	for _, obj := range bundle.Objects {
		if obj.Type == "attack-pattern" {
			for _, ref := range obj.ExternalReferences {
				if ref.SourceName == "mitre-attack" {
					mitreMap[obj.ID] = ref.ExternalID
				}
			}
		}
	}

	var out []NormalizedIndicator
	for _, obj := range bundle.Objects {
		if obj.Type != "indicator" {
			continue
		}
		iocType, iocValue := parsePattern(obj.Pattern)
		if iocType == "" || iocValue == "" {
			continue
		}

		severity := "medium"
		category := ""
		for _, l := range obj.Labels {
			l = strings.ToLower(l)
			switch {
			case strings.Contains(l, "malicious-activity"):
				severity = "high"
			case strings.Contains(l, "anomalous-activity"):
				severity = "medium"
			case strings.Contains(l, "benign"):
				severity = "low"
			}
			if strings.Contains(l, "apt") {
				category = "apt"
			} else if strings.Contains(l, "ransomware") {
				category = "ransomware"
			} else if strings.Contains(l, "phishing") {
				category = "phishing"
			}
		}

		conf := float64(obj.Confidence) / 100.0
		if conf <= 0 || conf > 1 {
			conf = 0.5
		}

		firstSeen := obj.ValidFrom
		if firstSeen.IsZero() {
			firstSeen = obj.Created
		}
		if firstSeen.IsZero() {
			firstSeen = time.Now().UTC()
		}

		// Collect MITRE technique IDs from kill chain phases
		var techniques []string
		for _, kc := range obj.KillChainPhases {
			if kc.KillChainName == "mitre-attack" {
				techniques = append(techniques, kc.PhaseName)
			}
		}

		ind := NormalizedIndicator{
			Title:           fmt.Sprintf("STIX Indicator: %s", truncate(obj.Name, 200)),
			Description:     obj.Description,
			SeverityCode:    severity,
			CategoryCode:    category,
			ConfidenceScore: conf,
			IOCType:         iocType,
			IOCValue:        iocValue,
			MITRETechniques: techniques,
			ExternalRef:     obj.ID,
			FirstSeen:       firstSeen,
			LastSeen:        time.Now().UTC(),
			Tags:            obj.Labels,
		}

		out = append(out, ind)
	}

	return out, nil
}

func parsePattern(pattern string) (string, string) {
	if m := ipv4Pattern.FindStringSubmatch(pattern); len(m) == 2 {
		return "ip", m[1]
	}
	if m := domainPattern.FindStringSubmatch(pattern); len(m) == 2 {
		return "domain", m[1]
	}
	if m := urlPattern.FindStringSubmatch(pattern); len(m) == 2 {
		return "url", m[1]
	}
	if m := hashPattern.FindStringSubmatch(pattern); len(m) == 3 {
		hashType := strings.ToLower(m[1])
		switch hashType {
		case "sha-256", "sha256":
			return "hash_sha256", m[2]
		case "md5":
			return "hash_md5", m[2]
		case "sha-1", "sha1":
			return "hash_sha1", m[2]
		}
		return "hash_sha256", m[2]
	}
	if m := emailPattern.FindStringSubmatch(pattern); len(m) == 2 {
		return "email", m[1]
	}
	return "", ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
