package handler

import (
	"testing"
	"time"
)

// TestIsBlocked_NotBlocked verifies fresh state is not blocked.
func TestIsBlocked_NotBlocked(t *testing.T) {
	app := newTestApp()
	ok, _ := app.isBlocked("1.2.3.4", "user1")
	if ok {
		t.Error("expected not blocked on fresh app")
	}
}

// TestIsBlocked_BlockedAfterUserFailures verifies user-based lock after threshold failures.
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

// TestIsBlocked_BlockedAfterIPFailures verifies IP-based lock after threshold failures.
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

// TestRecordFail_UnknownUserSkipsUserFails verifies unknown users do not increment user-specific tracking.
func TestRecordFail_UnknownUserSkipsUserFails(t *testing.T) {
	app := newTestApp()
	for i := 0; i < 10; i++ {
		app.recordFail("1.2.3.4", "unknownuser", false)
	}
	if _, ok := app.userFails["unknownuser"]; ok {
		t.Error("expected no userFails entry when trackUser=false")
	}
}

// TestRecordFail_KnownUserTracksUserFails verifies known users increment user-specific tracking.
func TestRecordFail_KnownUserTracksUserFails(t *testing.T) {
	app := newTestApp()
	app.recordFail("1.2.3.4", "knownuser", true)
	if _, ok := app.userFails["knownuser"]; !ok {
		t.Error("expected userFails entry when trackUser=true")
	}
}

// TestLockDuration verifies lock backoff schedule by lock count.
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

// TestResetWindow_FailCountAfter15Min verifies failure count reset after inactivity window.
func TestResetWindow_FailCountAfter15Min(t *testing.T) {
	app := newTestApp()
	app.userFails["user1"] = &loginAttempt{
		FailCount:  4,
		LastFailAt: time.Now().Add(-20 * time.Minute),
	}
	app.recordFail("1.2.3.4", "user1", true)
	s := app.userFails["user1"]
	if s.FailCount != 1 {
		t.Errorf("expected FailCount=1 after 15-min reset, got %d", s.FailCount)
	}
}

// TestResetWindow_LockCountAfter30Min verifies lock escalation reset after cooldown window.
func TestResetWindow_LockCountAfter30Min(t *testing.T) {
	app := newTestApp()
	app.userFails["user1"] = &loginAttempt{
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
