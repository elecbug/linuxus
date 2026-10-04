package subnet

import (
	"fmt"
	"net"
	"strings"
)

// IsValidSubnet checks if the given string is a valid subnet in CIDR notation.
func IsValidSubnet(subnet string) bool {
	ip, network, err := net.ParseCIDR(subnet)
	if err != nil {
		return false
	}
	_, bits := network.Mask.Size()
	return bits == 32 && ip.To4() != nil
}

// IsValidSubnetList checks if the given string is a valid comma-separated list of CIDR blocks.
// Empty string is considered valid (no proxies).
func IsValidSubnetList(proxies string) error {
	if proxies == "" {
		return nil // Empty is allowed
	}

	proxyList := strings.Split(proxies, ",")

	for _, proxy := range proxyList {
		proxy = strings.TrimSpace(proxy)
		if _, _, err := net.ParseCIDR(proxy); err != nil {
			return fmt.Errorf("invalid CIDR block: %s", proxy)
		}
	}

	return nil
}

// IsValidSubnet16 checks if the given string is a valid /16 subnet.
// Like IsValidSubnet but specifically for /16 subnets in the form x.x.0.0.
func IsValidSubnet16(subnet string) bool {
	ip := net.ParseIP(subnet).To4()
	return ip != nil && !strings.Contains(subnet, ":") && ip[2] == 0 && ip[3] == 0
}

// GetSubnetByIndex computes a /28 subnet string from base IP and slot index.
func GetSubnetByIndex(baseIP string, index int) (string, error) {
	ip := net.ParseIP(baseIP).To4()
	if !IsValidSubnet16(baseIP) {
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

// SubnetToIndex converts a /28 subnet back to a slot index relative to base IP.
func SubnetToIndex(baseIP, subnet string) (int, bool) {
	base := net.ParseIP(baseIP).To4()
	if !IsValidSubnet16(baseIP) {
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
