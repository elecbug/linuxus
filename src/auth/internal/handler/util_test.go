package handler

import (
	"testing"
	"time"
)

func newTestApp() *App {
	return &App{
		ipFails:   make(map[string]*LoginAttempt),
		userFails: make(map[string]*LoginAttempt),
	}
}

func TestIsBlocked_NotBlocked(t *testing.T) {
	app := newTestApp()
	ok, _ := app.isBlocked("1.2.3.4", "user1")
	if ok {
		t.Error("expected not blocked on fresh app")
	}
}

func TestIsBlocked_BlockedAfterUserFailures(t *testing.T) {
	app := newTestApp()
	for i := 0; i < 5; i++ {
		app.recordFail("1.2.3.4", "user1", true)
	}
	ok, until := app.isBlocked("1.2.3.4", "user1")
	if !ok {
		t.Error("expected user to be blocked after 5 failures")
	}
	if until.IsZero() {
		t.Error("expected non-zero lock time")
	}
}

func TestIsBlocked_BlockedAfterIPFailures(t *testing.T) {
	app := newTestApp()
	for i := 0; i < 20; i++ {
		app.recordFail("1.2.3.4", "user"+string(rune('A'+i)), false)
	}
	ok, _ := app.isBlocked("1.2.3.4", "anyuser")
	if !ok {
		t.Error("expected IP to be blocked after 20 failures")
	}
}

func TestRecordFail_UnknownUserSkipsUserFails(t *testing.T) {
	app := newTestApp()
	for i := 0; i < 10; i++ {
		app.recordFail("1.2.3.4", "unknownuser", false)
	}
	if _, ok := app.userFails["unknownuser"]; ok {
		t.Error("expected no userFails entry when trackUser=false")
	}
}

func TestRecordFail_KnownUserTracksUserFails(t *testing.T) {
	app := newTestApp()
	app.recordFail("1.2.3.4", "knownuser", true)
	if _, ok := app.userFails["knownuser"]; !ok {
		t.Error("expected userFails entry when trackUser=true")
	}
}

func TestLockDuration(t *testing.T) {
	cases := []struct {
		lockCount int
		expected  time.Duration
	}{
		{1, 1 * time.Minute},
		{2, 2 * time.Minute},
		{3, 4 * time.Minute},
		{4, 8 * time.Minute},
		{10, 8 * time.Minute},
	}
	for _, tc := range cases {
		got := lockDuration(tc.lockCount)
		if got != tc.expected {
			t.Errorf("lockDuration(%d) = %v, want %v", tc.lockCount, got, tc.expected)
		}
	}
}

func TestFailDelay_Default(t *testing.T) {
	app := newTestApp()
	d := app.failDelay("user1")
	if d != 300*time.Millisecond {
		t.Errorf("expected 300ms default delay, got %v", d)
	}
}

func TestFailDelay_AfterFailures(t *testing.T) {
	app := newTestApp()
	for i := 0; i < 3; i++ {
		app.recordFail("1.2.3.4", "user1", true)
	}
	d := app.failDelay("user1")
	if d != 3*300*time.Millisecond {
		t.Errorf("expected 900ms delay after 3 failures, got %v", d)
	}
}

func TestFailDelay_CappedAt5(t *testing.T) {
	app := newTestApp()
	app.userFails["user1"] = &LoginAttempt{FailCount: 10}
	d := app.failDelay("user1")
	if d != 5*300*time.Millisecond {
		t.Errorf("expected 1500ms (cap), got %v", d)
	}
}

func TestResetWindow_FailCountAfter15Min(t *testing.T) {
	app := newTestApp()
	app.userFails["user1"] = &LoginAttempt{
		FailCount:  4,
		LastFailAt: time.Now().Add(-20 * time.Minute),
	}
	app.recordFail("1.2.3.4", "user1", true)
	s := app.userFails["user1"]
	if s.FailCount != 1 {
		t.Errorf("expected FailCount=1 after 15-min reset, got %d", s.FailCount)
	}
}

func TestResetWindow_LockCountAfter30Min(t *testing.T) {
	app := newTestApp()
	app.userFails["user1"] = &LoginAttempt{
		LockCount:   3,
		LockedUntil: time.Now().Add(-35 * time.Minute),
		LastFailAt:  time.Now().Add(-35 * time.Minute),
	}
	app.recordFail("1.2.3.4", "user1", true)
	s := app.userFails["user1"]
	if s.LockCount != 0 {
		t.Errorf("expected LockCount=0 after 30-min reset, got %d", s.LockCount)
	}
}

func TestEvictStaleEntries(t *testing.T) {
	app := newTestApp()

	// Stale: last fail >30 min ago, not locked
	app.ipFails["stale"] = &LoginAttempt{
		LastFailAt: time.Now().Add(-31 * time.Minute),
	}
	// Active: failed recently
	app.ipFails["active"] = &LoginAttempt{
		LastFailAt: time.Now(),
	}
	// Old but still locked: should not be evicted
	app.ipFails["locked"] = &LoginAttempt{
		LastFailAt:  time.Now().Add(-31 * time.Minute),
		LockedUntil: time.Now().Add(5 * time.Minute),
	}

	app.evictStaleEntries()

	if _, ok := app.ipFails["stale"]; ok {
		t.Error("stale entry should have been evicted")
	}
	if _, ok := app.ipFails["active"]; !ok {
		t.Error("active entry should not have been evicted")
	}
	if _, ok := app.ipFails["locked"]; !ok {
		t.Error("still-locked entry should not have been evicted")
	}
}

func TestParseTrustedProxies_Empty(t *testing.T) {
	result := ParseTrustedProxies("")
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty input, got %v", result)
	}
}

func TestParseTrustedProxies_SingleCIDR(t *testing.T) {
	result := ParseTrustedProxies("192.168.1.0/24")
	if len(result) != 1 || result[0] != "192.168.1.0/24" {
		t.Errorf("expected single CIDR entry, got %v", result)
	}
}

func TestParseTrustedProxies_CommaSeparated(t *testing.T) {
	result := ParseTrustedProxies("10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")
	if len(result) != 3 {
		t.Errorf("expected 3 entries, got %d: %v", len(result), result)
	}
}

func TestParseTrustedProxies_TrimsWhitespace(t *testing.T) {
	result := ParseTrustedProxies("  10.0.0.0/8 , 172.16.0.0/12  ")
	if len(result) != 2 {
		t.Errorf("expected 2 entries after trimming, got %d: %v", len(result), result)
	}
	if result[0] != "10.0.0.0/8" || result[1] != "172.16.0.0/12" {
		t.Errorf("unexpected values after trim: %v", result)
	}
}

func TestParseTrustedProxies_SkipsEmptyEntries(t *testing.T) {
	result := ParseTrustedProxies("10.0.0.0/8,,192.168.0.0/16")
	if len(result) != 2 {
		t.Errorf("expected 2 entries (skipping empty), got %d: %v", len(result), result)
	}
}
