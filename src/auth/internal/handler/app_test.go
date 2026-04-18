package handler

import (
	"testing"
	"time"
)

// newTestApp creates an App with initialized failure tracking maps for tests.
func newTestApp() *App {
	return &App{
		ipFails:   make(map[string]*loginAttempt),
		userFails: make(map[string]*loginAttempt),
	}
}

// TestEvictStaleEntries verifies stale failure entries are cleaned up correctly.
func TestEvictStaleEntries(t *testing.T) {
	app := newTestApp()

	// Stale: last fail >30 min ago, not locked
	app.ipFails["stale"] = &loginAttempt{
		LastFailAt: time.Now().Add(-31 * time.Minute),
	}
	// Active: failed recently
	app.ipFails["active"] = &loginAttempt{
		LastFailAt: time.Now(),
	}
	// Old but still locked: should not be evicted
	app.ipFails["locked"] = &loginAttempt{
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
