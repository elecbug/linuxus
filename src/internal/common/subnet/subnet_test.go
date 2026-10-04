package subnet

import "testing"

func TestSubnetParsingMatchesRuntime(t *testing.T) {
	for _, value := range []string{"010.10.0.0/16", "::ffff:10.10.0.0/112", "256.1.0.0/16", "10.1.0.0/33", "not-a-subnet"} {
		if IsValidSubnet(value) {
			t.Errorf("accepted subnet %q", value)
		}
	}
	for _, base := range []string{"010.10.0.0", "10.10.1.0", "10.10.0.1", "::ffff:10.10.0.0", "invalid"} {
		if IsValidSubnet16(base) {
			t.Errorf("accepted base %q", base)
		}
		if _, err := GetSubnetByIndex(base, 0); err == nil {
			t.Errorf("allocated from base %q", base)
		}
		if _, ok := SubnetToIndex(base, "10.10.0.0/28"); ok {
			t.Errorf("accepted reverse base %q", base)
		}
	}
	if err := IsValidSubnetList("10.0.0.0/8, 2001:db8::/32, ::1/128"); err != nil {
		t.Fatalf("rejected trusted IPv6 proxy: %v", err)
	}
	if err := IsValidSubnetList("10.0.0.0/8,invalid"); err == nil {
		t.Fatal("accepted invalid proxy")
	}
}

func TestEveryUserSubnetRoundTripsAndStaysWithinBase(t *testing.T) {
	seen := make(map[string]bool)
	for index := 0; index < 4096; index++ {
		sn, err := GetSubnetByIndex("10.10.0.0", index)
		if err != nil {
			t.Fatal(err)
		}
		if seen[sn] {
			t.Fatalf("duplicate subnet %s", sn)
		}
		seen[sn] = true
		got, ok := SubnetToIndex("10.10.0.0", sn)
		if !ok || got != index {
			t.Fatalf("slot %d maps back to %d/%t", index, got, ok)
		}
	}
	for _, index := range []int{-1, 4096} {
		if _, err := GetSubnetByIndex("10.10.0.0", index); err == nil {
			t.Errorf("accepted slot %d outside /16", index)
		}
	}
	for _, sn := range []string{"10.11.0.0/28", "10.10.0.1/28", "10.10.0.0/24"} {
		if _, ok := SubnetToIndex("10.10.0.0", sn); ok {
			t.Errorf("accepted invalid user subnet %q", sn)
		}
	}
}
