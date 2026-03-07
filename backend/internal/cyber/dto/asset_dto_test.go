package dto

import (
	"strings"
	"testing"

	"github.com/clario360/platform/internal/cyber/model"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

func TestCreateAssetValidation(t *testing.T) {
	validIP := "10.0.1.5"
	validMAC := "aa:bb:cc:dd:ee:ff"

	req := CreateAssetRequest{
		Name:        "web-prod-01",
		Type:        model.AssetTypeServer,
		IPAddress:   &validIP,
		MACAddress:  &validMAC,
		Criticality: model.CriticalityHigh,
		Tags:        []string{"production", "web"},
	}
	if errs := pkgvalidator.Validate(req); errs != nil {
		t.Fatalf("expected valid request, got %#v", errs)
	}

	invalidReq := CreateAssetRequest{Name: "", Type: model.AssetType("invalid")}
	errs := pkgvalidator.Validate(invalidReq)
	if errs["name"] == "" || errs["type"] == "" {
		t.Fatalf("expected validation errors, got %#v", errs)
	}
}

func TestCreateAssetValidation_InvalidMACAndTags(t *testing.T) {
	invalidMAC := "invalid"
	req := CreateAssetRequest{
		Name:        "asset-1",
		Type:        model.AssetTypeServer,
		MACAddress:  &invalidMAC,
		Criticality: model.CriticalityLow,
		Tags:        make([]string, 21),
	}
	errs := pkgvalidator.Validate(req)
	if errs["mac_address"] == "" || errs["tags"] == "" {
		t.Fatalf("expected mac and tags errors, got %#v", errs)
	}

	req = CreateAssetRequest{
		Name:        "asset-1",
		Type:        model.AssetTypeServer,
		Criticality: model.CriticalityLow,
		Tags:        []string{"bad tag!"},
	}
	errs = pkgvalidator.Validate(req)
	if !strings.Contains(errs["tags[0]"], "letters") && errs["tags"] == "" {
		t.Fatalf("expected invalid tag error, got %#v", errs)
	}
}

func TestCreateAssetValidation_InvalidIP(t *testing.T) {
	invalidIP := "not-an-ip"
	req := CreateAssetRequest{
		Name:        "asset-1",
		Type:        model.AssetTypeServer,
		IPAddress:   &invalidIP,
		Criticality: model.CriticalityLow,
	}
	errMap := pkgvalidator.Validate(req)
	if errMap["ip_address"] == "" {
		t.Fatalf("expected ip validation error, got %#v", errMap)
	}
}

func TestAssetListParams_DefaultsAndValidate(t *testing.T) {
	params := &AssetListParams{}
	params.SetDefaults()
	if params.Page != 1 || params.PerPage != 25 || params.Sort != "created_at" || params.Order != "desc" {
		t.Fatalf("unexpected defaults: %#v", params)
	}

	params = &AssetListParams{
		Types:         []string{"server"},
		Criticalities: []string{"high"},
		Statuses:      []string{"active"},
		Tags:          []string{"production"},
		Sort:          "created_at",
		Order:         "asc",
	}
	if err := params.Validate(); err != nil {
		t.Fatalf("expected params to validate, got %v", err)
	}

	params = &AssetListParams{Types: []string{"DROP TABLE"}}
	if err := params.Validate(); err == nil {
		t.Fatal("expected invalid type error")
	}

	params = &AssetListParams{Sort: "DROP TABLE"}
	if err := params.Validate(); err == nil {
		t.Fatal("expected invalid sort error")
	}

	params = &AssetListParams{Tags: []string{"bad tag!"}}
	if err := params.Validate(); err == nil {
		t.Fatal("expected invalid tag filter error")
	}

	params = &AssetListParams{PerPage: 500}
	params.SetDefaults()
	if params.PerPage != 200 {
		t.Fatalf("expected per-page clamp to 200, got %d", params.PerPage)
	}
}
