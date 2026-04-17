package handler

import (
	"testing"
)

// TestParseTrustedProxies_Empty verifies empty input produces no CIDR entries.
func TestParseTrustedProxies_Empty(t *testing.T) {
	result := ParseTrustedProxies("")
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty input, got %v", result)
	}
}

// TestParseTrustedProxies_SingleCIDR verifies one CIDR token is parsed correctly.
func TestParseTrustedProxies_SingleCIDR(t *testing.T) {
	result := ParseTrustedProxies("192.168.1.0/24")
	if len(result) != 1 || result[0] != "192.168.1.0/24" {
		t.Errorf("expected single CIDR entry, got %v", result)
	}
}

// TestParseTrustedProxies_CommaSeparated verifies multiple CIDRs are parsed in order.
func TestParseTrustedProxies_CommaSeparated(t *testing.T) {
	result := ParseTrustedProxies("10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")
	if len(result) != 3 {
		t.Errorf("expected 3 entries, got %d: %v", len(result), result)
	}
}

// TestParseTrustedProxies_TrimsWhitespace verifies surrounding whitespace is ignored.
func TestParseTrustedProxies_TrimsWhitespace(t *testing.T) {
	result := ParseTrustedProxies("  10.0.0.0/8 , 172.16.0.0/12  ")
	if len(result) != 2 {
		t.Errorf("expected 2 entries after trimming, got %d: %v", len(result), result)
	}
	if result[0] != "10.0.0.0/8" || result[1] != "172.16.0.0/12" {
		t.Errorf("unexpected values after trim: %v", result)
	}
}

// TestParseTrustedProxies_SkipsEmptyEntries verifies empty comma tokens are ignored.
func TestParseTrustedProxies_SkipsEmptyEntries(t *testing.T) {
	result := ParseTrustedProxies("10.0.0.0/8,,192.168.0.0/16")
	if len(result) != 2 {
		t.Errorf("expected 2 entries (skipping empty), got %d: %v", len(result), result)
	}
}
