package scanner

import (
	"net"
	"testing"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestExpandCIDR(t *testing.T) {
	ips, err := expandCIDR("10.0.1.0/24", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 254 {
		t.Fatalf("expected 254 ips, got %d", len(ips))
	}

	ips, err = expandCIDR("10.0.1.5/32", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 1 || !ips[0].Equal(net.ParseIP("10.0.1.5")) {
		t.Fatalf("unexpected /32 expansion: %#v", ips)
	}
}

func TestExpandCIDR_PublicBlockedAndTooLarge(t *testing.T) {
	if _, err := expandCIDR("8.8.8.0/24", false); err == nil {
		t.Fatal("expected public IP scan to be blocked")
	}
	if _, err := expandCIDR("10.0.0.0/7", false); err == nil {
		t.Fatal("expected oversized cidr to be rejected")
	}
	if _, err := expandCIDR("not-a-cidr", false); err == nil {
		t.Fatal("expected invalid cidr to be rejected")
	}
}

func TestExpandCIDR_16(t *testing.T) {
	ips, err := expandCIDR("10.1.0.0/16", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 65534 {
		t.Fatalf("expected 65534 IPs, got %d", len(ips))
	}
}

func TestExpandCIDRs_MaxIPs(t *testing.T) {
	if _, err := expandCIDRs([]string{"10.0.0.0/24", "10.0.1.0/24"}, 300, false); err == nil {
		t.Fatal("expected max ip limit error")
	}
}

func TestInferAssetType(t *testing.T) {
	if got := inferAssetType([]int{22, 80, 443}); got != model.AssetTypeServer {
		t.Fatalf("expected server, got %s", got)
	}
	if got := inferAssetType([]int{3389}); got != model.AssetTypeEndpoint {
		t.Fatalf("expected endpoint, got %s", got)
	}
	if got := inferAssetType([]int{161}); got != model.AssetTypeNetworkDevice {
		t.Fatalf("expected network device, got %s", got)
	}
	if got := inferAssetType([]int{5432}); got != model.AssetTypeDatabase {
		t.Fatalf("expected database, got %s", got)
	}
}

func TestInferOS(t *testing.T) {
	osName, version := inferOS(map[int]string{22: "SSH-2.0-OpenSSH_8.9p1 Ubuntu-3"})
	if osName == nil || *osName != "linux" {
		t.Fatalf("expected linux, got %#v", osName)
	}
	if version == nil || *version != "Ubuntu" {
		t.Fatalf("expected Ubuntu version, got %#v", version)
	}

	osName, version = inferOS(map[int]string{80: "Server: Microsoft-IIS/10.0"})
	if osName == nil || *osName != "windows" {
		t.Fatalf("expected windows, got %#v", osName)
	}
	if version == nil || *version != "IIS" {
		t.Fatalf("expected IIS, got %#v", version)
	}
}
