package convert

import "testing"

func TestResourceLimitsRejectOverflow(t *testing.T) {
	for _, value := range []string{"8589934592G", "18014398509481985k", "9223372036854775807m"} {
		if got, err := BytesFromString(value); err == nil {
			t.Errorf("size %q overflowed to %d", value, got)
		}
	}
	for _, value := range []string{"NaN", "+Inf", "-Inf", "1e20", "9223372037", "-0.5"} {
		if got, err := NanoCPUsFromString(value); err == nil {
			t.Errorf("CPU %q accepted as %d", value, got)
		}
	}
	if got, err := BytesFromString("2GB"); err != nil || got != 2*1024*1024*1024 {
		t.Fatalf("valid size: %d, %v", got, err)
	}
	if got, err := NanoCPUsFromString("1.5"); err != nil || got != 1500000000 {
		t.Fatalf("valid CPU: %d, %v", got, err)
	}
}

func TestPortBindingBounds(t *testing.T) {
	for _, value := range []string{"bad:80", "65536:80", "-1:80", ":80", "80:0", "80:65536", "80:-1"} {
		if _, _, err := PortBindingFromString(value); err == nil {
			t.Errorf("accepted port %q", value)
		}
	}
	for _, value := range []string{"0:80", "8080:80", "65535:65535"} {
		if _, _, err := PortBindingFromString(value); err != nil {
			t.Errorf("rejected %q: %v", value, err)
		}
	}
}
