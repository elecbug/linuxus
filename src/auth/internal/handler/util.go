package handler

import "strings"

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
