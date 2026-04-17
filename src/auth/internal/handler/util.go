package handler

import "strings"

// ParseTrustedProxies parses a comma-separated trusted proxy CIDR list.
func ParseTrustedProxies(trustedProxies string) []string {
	var trustedProxyCIDRs []string

	if tp := trustedProxies; tp != "" {
		for _, cidr := range strings.Split(tp, ",") {
			cidr = strings.TrimSpace(cidr)
			if cidr != "" {
				trustedProxyCIDRs = append(trustedProxyCIDRs, cidr)
			}
		}
	}

	return trustedProxyCIDRs
}
