package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// DependencyChecker performs runtime dependency vulnerability checks.
// It gracefully handles missing tools (govulncheck, npm) by reporting
// them as scan errors rather than failing.
type DependencyChecker struct {
	logger  zerolog.Logger
	timeout time.Duration
}

// NewDependencyChecker creates a new dependency checker.
func NewDependencyChecker(logger zerolog.Logger) *DependencyChecker {
	return &DependencyChecker{
		logger:  logger.With().Str("component", "dependency_check").Logger(),
		timeout: 2 * time.Minute,
	}
}

// VulnerabilityReport contains the results of a dependency vulnerability scan.
type VulnerabilityReport struct {
	Timestamp     time.Time         `json:"timestamp"`
	GoVulns       []GoVulnerability `json:"go_vulnerabilities,omitempty"`
	NPMVulns      []NPMVulnerability `json:"npm_vulnerabilities,omitempty"`
	TotalCritical int               `json:"total_critical"`
	TotalHigh     int               `json:"total_high"`
	TotalMedium   int               `json:"total_medium"`
	TotalLow      int               `json:"total_low"`
	ScanErrors    []string          `json:"scan_errors,omitempty"`
	ToolVersions  map[string]string `json:"tool_versions,omitempty"`
}

// GoVulnerability represents a Go module vulnerability.
type GoVulnerability struct {
	ID          string `json:"id"`
	Module      string `json:"module"`
	Version     string `json:"version"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	FixedIn     string `json:"fixed_in,omitempty"`
}

// NPMVulnerability represents an npm package vulnerability.
type NPMVulnerability struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Severity     string `json:"severity"`
	Title        string `json:"title"`
	URL          string `json:"url,omitempty"`
	FixAvailable bool   `json:"fix_available"`
}

// CheckToolAvailability verifies which scanning tools are installed and returns
// their versions. Missing tools are reported but do not cause errors.
func (dc *DependencyChecker) CheckToolAvailability() map[string]string {
	versions := make(map[string]string)

	// Check govulncheck
	if version, err := dc.getToolVersion("govulncheck", "-version"); err == nil {
		versions["govulncheck"] = version
	} else {
		dc.logger.Warn().Msg("govulncheck not found — install with: go install golang.org/x/vuln/cmd/govulncheck@latest")
	}

	// Check npm
	if version, err := dc.getToolVersion("npm", "--version"); err == nil {
		versions["npm"] = version
	} else {
		dc.logger.Warn().Msg("npm not found — install Node.js from https://nodejs.org/")
	}

	return versions
}

// getToolVersion runs a tool with a version flag and returns trimmed output.
func (dc *DependencyChecker) getToolVersion(tool, flag string) (string, error) {
	path, err := exec.LookPath(tool)
	if err != nil {
		return "", fmt.Errorf("%s not in PATH: %w", tool, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, flag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s failed: %w", tool, flag, err)
	}

	version := strings.TrimSpace(string(output))
	// Take only the first line (some tools print multi-line version info)
	if idx := strings.IndexByte(version, '\n'); idx > 0 {
		version = version[:idx]
	}
	return version, nil
}

// CheckGoVulnerabilities runs govulncheck on the Go module.
// Returns nil with a descriptive error if govulncheck is not installed.
func (dc *DependencyChecker) CheckGoVulnerabilities(ctx context.Context, modulePath string) ([]GoVulnerability, error) {
	govulncheck, err := exec.LookPath("govulncheck")
	if err != nil {
		return nil, fmt.Errorf("govulncheck not installed — install with: go install golang.org/x/vuln/cmd/govulncheck@latest")
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, govulncheck, "-json", "./...")
	cmd.Dir = modulePath
	cmd.Env = append(cmd.Environ(), "GOWORK=off")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// govulncheck exits non-zero when vulns are found — that's expected
		if ctx.Err() != nil {
			return nil, fmt.Errorf("govulncheck timed out after %s", dc.timeout)
		}
	}

	return parseGovulncheckOutput(output)
}

// CheckNPMVulnerabilities runs npm audit on the frontend.
// Returns nil with a descriptive error if npm is not installed.
func (dc *DependencyChecker) CheckNPMVulnerabilities(ctx context.Context, projectPath string) ([]NPMVulnerability, error) {
	npmPath, err := exec.LookPath("npm")
	if err != nil {
		return nil, fmt.Errorf("npm not installed — install Node.js from https://nodejs.org/")
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, npmPath, "audit", "--json")
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("npm audit timed out after %s", dc.timeout)
		}
	}

	return parseNPMAuditOutput(output)
}

// RunFullScan runs both Go and npm vulnerability scans.
// Missing tools are reported in ScanErrors but do not prevent other scans.
func (dc *DependencyChecker) RunFullScan(ctx context.Context, goModulePath, npmProjectPath string) *VulnerabilityReport {
	report := &VulnerabilityReport{
		Timestamp:    time.Now().UTC(),
		ToolVersions: dc.CheckToolAvailability(),
	}

	// Go vulnerabilities
	if goModulePath != "" {
		goVulns, err := dc.CheckGoVulnerabilities(ctx, goModulePath)
		if err != nil {
			report.ScanErrors = append(report.ScanErrors, "go: "+err.Error())
			dc.logger.Warn().Err(err).Msg("Go vulnerability scan skipped or failed")
		} else {
			report.GoVulns = goVulns
		}
	}

	// NPM vulnerabilities
	if npmProjectPath != "" {
		npmVulns, err := dc.CheckNPMVulnerabilities(ctx, npmProjectPath)
		if err != nil {
			report.ScanErrors = append(report.ScanErrors, "npm: "+err.Error())
			dc.logger.Warn().Err(err).Msg("NPM vulnerability scan skipped or failed")
		} else {
			report.NPMVulns = npmVulns
		}
	}

	// Count by severity
	for _, v := range report.GoVulns {
		countSeverity(v.Severity, report)
	}
	for _, v := range report.NPMVulns {
		countSeverity(v.Severity, report)
	}

	return report
}

func countSeverity(severity string, report *VulnerabilityReport) {
	switch strings.ToLower(severity) {
	case "critical":
		report.TotalCritical++
	case "high":
		report.TotalHigh++
	case "moderate", "medium":
		report.TotalMedium++
	case "low":
		report.TotalLow++
	}
}

// parseGovulncheckOutput parses govulncheck JSON output.
func parseGovulncheckOutput(output []byte) ([]GoVulnerability, error) {
	var vulns []GoVulnerability

	// govulncheck outputs newline-delimited JSON objects
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Look for vulnerability findings
		if finding, ok := entry["finding"].(map[string]interface{}); ok {
			vuln := GoVulnerability{}
			if osv, ok := finding["osv"].(string); ok {
				vuln.ID = osv
			}
			if trace, ok := finding["trace"].([]interface{}); ok && len(trace) > 0 {
				if first, ok := trace[0].(map[string]interface{}); ok {
					if module, ok := first["module"].(string); ok {
						vuln.Module = module
					}
					if version, ok := first["version"].(string); ok {
						vuln.Version = version
					}
				}
			}
			if fixedVersion, ok := finding["fixed_version"].(string); ok {
				vuln.FixedIn = fixedVersion
			}
			vuln.Severity = "high" // govulncheck doesn't provide severity directly
			if vuln.ID != "" {
				vulns = append(vulns, vuln)
			}
		}
	}

	return vulns, nil
}

// parseNPMAuditOutput parses npm audit JSON output.
func parseNPMAuditOutput(output []byte) ([]NPMVulnerability, error) {
	var result struct {
		Vulnerabilities map[string]struct {
			Name         string        `json:"name"`
			Severity     string        `json:"severity"`
			Via          []interface{} `json:"via"`
			Range        string        `json:"range"`
			FixAvailable interface{}   `json:"fixAvailable"`
		} `json:"vulnerabilities"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse npm audit output: %w", err)
	}

	var vulns []NPMVulnerability
	for name, v := range result.Vulnerabilities {
		npm := NPMVulnerability{
			Name:         name,
			Version:      v.Range,
			Severity:     v.Severity,
			FixAvailable: v.FixAvailable != nil && v.FixAvailable != false,
		}

		// Extract title from "via" entries
		for _, via := range v.Via {
			if viaMap, ok := via.(map[string]interface{}); ok {
				if title, ok := viaMap["title"].(string); ok {
					npm.Title = title
					break
				}
			}
		}

		vulns = append(vulns, npm)
	}

	return vulns, nil
}
