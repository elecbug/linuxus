package main

import "testing"

func TestRejectAmbiguousArguments(t *testing.T) {
	for _, args := range [][]string{
		{"ensure-disk", "--user"}, {"clean-volume", "--user"}, {"remove-user", "--user"},
		{"clean-volume", "--user", "alice", "--user", "bob"},
		{"ensure-disk", "--user", "alice", "-u", "bob"},
		{"ensure-disk", "--all", "--user", "alice"},
		{"ps", "unknown"}, {"ps", "container", "extra"}, {"init", "extra"}, {"init", "--all"}, {"up", "extra"}, {"help", "extra"},
		{"serve-disks", "--all"}, {"add-user", "--unknown", "alice"},
		{"clean-volume", "--all=false"}, {"add-user", "--user", "../outside"},
	} {
		if _, err := parseArgs("linuxusctl", args); err == nil {
			t.Errorf("accepted %q", args)
		}
	}
}

func TestValidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"init"}, {"up"}, {"down"}, {"restart"}, {"help"}, {"ps"}, {"ps", "n"}, {"ps", "NETWORK"},
		{"ensure-disk", "--all"}, {"clean-volume", "-a"},
		{"add-user", "--user", "true"}, {"remove-user", "-u", "alice"},
		{"ensure-disk", "--user=alice"},
	} {
		if _, err := parseArgs("linuxusctl", args); err != nil {
			t.Errorf("rejected %q: %v", args, err)
		}
	}
}
