package detector

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type newSourceIPDetector struct {
	resolver GeoResolver
}

func (d *newSourceIPDetector) Name() model.SignalType {
	return model.SignalTypeNewSourceIP
}

func (d *newSourceIPDetector) Detect(ctx context.Context, event *model.DataAccessEvent, profile *model.UEBAProfile) *model.AnomalySignal {
	if strings.TrimSpace(event.SourceIP) == "" {
		return nil
	}
	for _, ip := range profile.Baseline.SourceIPs {
		if ip == event.SourceIP {
			return nil
		}
	}
	parsed := net.ParseIP(event.SourceIP)
	if parsed == nil {
		return nil
	}
	if profile.EntityType == model.EntityTypeServiceAccount && parsed.IsPrivate() {
		return nil
	}

	severity := "medium"
	confidence := 0.3
	expected := "known source IP range"
	actual := event.SourceIP

	current24 := cidrPrefix(parsed, 24)
	current16 := cidrPrefix(parsed, 16)
	match24 := false
	match16 := false
	for _, knownIP := range profile.Baseline.SourceIPs {
		parsedKnown := net.ParseIP(knownIP)
		if parsedKnown == nil {
			continue
		}
		if cidrPrefix(parsedKnown, 24) == current24 {
			match24 = true
		}
		if cidrPrefix(parsedKnown, 16) == current16 {
			match16 = true
		}
	}
	switch {
	case !match16:
		severity = "high"
		confidence = 0.6
	case match16 && !match24:
		severity = "medium"
		confidence = 0.45
	}

	if d.resolver != nil {
		currentCountry, err := d.resolver.Country(ctx, event.SourceIP)
		if err == nil && currentCountry != "" {
			for _, knownIP := range profile.Baseline.SourceIPs {
				knownCountry, lookupErr := d.resolver.Country(ctx, knownIP)
				if lookupErr != nil || knownCountry == "" {
					continue
				}
				if knownCountry != currentCountry {
					severity = "critical"
					confidence = 0.9
					expected = "known country of origin"
					actual = fmt.Sprintf("%s from %s", event.SourceIP, currentCountry)
					break
				}
			}
		}
	}

	return &model.AnomalySignal{
		SignalType:     d.Name(),
		Title:          "Unknown source IP",
		Description:    "The entity connected from an IP address outside its learned source set.",
		Severity:       severity,
		Confidence:     confidence,
		ExpectedValue:  expected,
		ActualValue:    actual,
		EventID:        event.ID,
		MITRETechnique: "T1078",
		MITRETactic:    "TA0006",
	}
}

func cidrPrefix(ip net.IP, maskBits int) string {
	if ip == nil {
		return ""
	}
	if ip.To4() != nil {
		ip = ip.To4()
	}
	mask := net.CIDRMask(maskBits, len(ip)*8)
	return ip.Mask(mask).String()
}
