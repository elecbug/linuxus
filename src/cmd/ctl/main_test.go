package main

import "testing"

func TestRejectAmbiguousArguments(t *testing.T) {
	for _, args := range [][]string{
		{"ensure-disk", "--user"}, {"ensure-disk", "--file", "anything"},
		{"backup-user", "--user", "alice"}, {"restore-user", "--user", "alice", "--file", "a.tar.gz", "--all"},
		{"assign-class", "--user", "alice"}, {"lock-user", "--all"}, {"verify-backup", "--user", "alice", "--file", "a"}, {"clean-volume", "--user"}, {"remove-user", "--user"},
		{"clean-volume", "--user", "alice", "--user", "bob"},
		{"ensure-disk", "--user", "alice", "-u", "bob"},
		{"ensure-disk", "--all", "--user", "alice"},
		{"config-check", "--all"}, {"doctor", "extra"}, {"ps", "unknown"}, {"ps", "container", "extra"}, {"init", "extra"}, {"init", "--all"}, {"up", "extra"}, {"help", "extra"},
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
		{"init"}, {"systemd-unit"}, {"list-users"}, {"templates"},
		{"lock-user", "--user", "alice"}, {"recover-user", "--user", "alice"},
		{"assign-class", "--user", "alice", "--class", "class-a"},
		{"backup-user", "--user", "alice", "--output", "/backups/a.tar.gz"},
		{"restore-user", "--user", "alice", "--file", "/backups/a.tar.gz", "--replace"},
		{"verify-backup", "--file", "a.tar.gz"}, {"config-check"}, {"doctor"}, {"up"}, {"down"}, {"restart"}, {"help"}, {"ps"}, {"ps", "n"}, {"ps", "NETWORK"},
		{"ensure-disk", "--all"}, {"clean-volume", "-a"},
		{"add-user", "--user", "true"}, {"remove-user", "-u", "alice"},
		{"ensure-disk", "--user=alice"},
	} {
		if _, err := parseArgs("linuxusctl", args); err != nil {
			t.Errorf("rejected %q: %v", args, err)
		}
	}
}
