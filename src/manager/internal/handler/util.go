package handler

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// getSubnetByIndex computes a /28 subnet string from base IP and slot index.
func getSubnetByIndex(baseIP string, index int) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(baseIP)).To4()
	if ip == nil {
		return "", fmt.Errorf("invalid base ip")
	}
	if index < 0 {
		return "", fmt.Errorf("invalid index")
	}

	o0, o1, o2 := int(ip[0]), int(ip[1]), int(ip[2])
	thirdOffset := index / 16
	fourthOffset := (index % 16) * 16

	newO2 := o2 + thirdOffset
	if newO2 > 255 {
		return "", fmt.Errorf("subnet overflow")
	}

	return fmt.Sprintf("%d.%d.%d.%d/28", o0, o1, newO2, fourthOffset), nil
}

// subnetToIndex converts a /28 subnet back to a slot index relative to base IP.
func subnetToIndex(baseIP, subnet string) (int, bool) {
	base := net.ParseIP(strings.TrimSpace(baseIP)).To4()
	if base == nil {
		return 0, false
	}

	ip, ipNet, err := net.ParseCIDR(strings.TrimSpace(subnet))
	if err != nil {
		return 0, false
	}
	ip = ip.To4()
	if ip == nil {
		return 0, false
	}

	ones, bits := ipNet.Mask.Size()
	if bits != 32 || ones != 28 {
		return 0, false
	}

	if ip[0] != base[0] || ip[1] != base[1] {
		return 0, false
	}
	if ip[3]%16 != 0 {
		return 0, false
	}
	if ip[2] < base[2] {
		return 0, false
	}

	thirdDelta := int(ip[2]) - int(base[2])
	index := thirdDelta*16 + int(ip[3])/16
	return index, true
}

// allowID checks if the provided ID is valid according to defined rules.
func allowID(id string) bool {
	if id == "" {
		return false
	}
	for i, ch := range id {
		if i == 0 && (ch == '_' || ch == '-') {
			return false
		}
		if i == len(id)-1 && (ch == '_' || ch == '-') {
			return false
		}
		if (ch >= 'A' && ch <= 'Z') ||
			(ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}

// writeJSON writes a JSON response with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
