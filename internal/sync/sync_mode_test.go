package sync

import (
	"testing"
	"time"
)

// SyncModeDecision represents the logic for deciding sync mode
type SyncModeDecision struct {
	ForceFullSync     bool
	LastSyncTime      time.Time
	LastFullSyncTime  time.Time
	FullSyncInterval  time.Duration
	LoadSyncTimeError error
}

// Decide returns the appropriate sync mode
// currentTime allows for deterministic testing
func (d *SyncModeDecision) Decide(currentTime time.Time) string {
	if d.ForceFullSync {
		return "full"
	}

	if d.LoadSyncTimeError != nil {
		return "full"
	}

	if d.LastSyncTime.IsZero() {
		return "full"
	}

	if !d.LastFullSyncTime.IsZero() && currentTime.Sub(d.LastFullSyncTime) > d.FullSyncInterval {
		return "full"
	}

	return "incremental"
}

func TestSyncModeDecision_ForceFullSync(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	decision := SyncModeDecision{
		ForceFullSync:    true,
		LastSyncTime:     now.Add(-1 * time.Hour),
		LastFullSyncTime: now.Add(-2 * 24 * time.Hour),
		FullSyncInterval: 7 * 24 * time.Hour,
	}

	mode := decision.Decide(now)
	if mode != "full" {
		t.Errorf("ForceFullSync should return 'full', got: %s", mode)
	}
}

func TestSyncModeDecision_FirstSync(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	decision := SyncModeDecision{
		ForceFullSync:    false,
		LastSyncTime:     time.Time{}, // Zero time = first sync
		LastFullSyncTime: time.Time{},
		FullSyncInterval: 7 * 24 * time.Hour,
	}

	mode := decision.Decide(now)
	if mode != "full" {
		t.Errorf("First sync should return 'full', got: %s", mode)
	}
}

func TestSyncModeDecision_RecentIncremental(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	decision := SyncModeDecision{
		ForceFullSync:    false,
		LastSyncTime:     now.Add(-1 * time.Hour),      // 1 hour ago
		LastFullSyncTime: now.Add(-2 * 24 * time.Hour), // 2 days ago
		FullSyncInterval: 7 * 24 * time.Hour,           // 7 days
	}

	mode := decision.Decide(now)
	if mode != "incremental" {
		t.Errorf("Recent sync should return 'incremental', got: %s", mode)
	}
}

func TestSyncModeDecision_AutoFullSync7Days(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	decision := SyncModeDecision{
		ForceFullSync:    false,
		LastSyncTime:     now.Add(-1 * time.Hour),      // 1 hour ago
		LastFullSyncTime: now.Add(-8 * 24 * time.Hour), // 8 days ago - should trigger full
		FullSyncInterval: 7 * 24 * time.Hour,           // 7 days
	}

	mode := decision.Decide(now)
	if mode != "full" {
		t.Errorf("8 days since full sync should return 'full', got: %s", mode)
	}
}

func TestSyncModeDecision_ExactlySevenDays(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	decision := SyncModeDecision{
		ForceFullSync:    false,
		LastSyncTime:     now.Add(-1 * time.Hour),
		LastFullSyncTime: now.Add(-7 * 24 * time.Hour), // Exactly 7 days
		FullSyncInterval: 7 * 24 * time.Hour,
	}

	mode := decision.Decide(now)
	// Exactly 7 days: current - last = 7 days, which is NOT > 7 days
	// So should be incremental
	if mode != "incremental" {
		t.Errorf("Exactly 7 days should return 'incremental', got: %s", mode)
	}
}

func TestSyncModeDecision_SevenDaysPlus1Second(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	decision := SyncModeDecision{
		ForceFullSync:    false,
		LastSyncTime:     now.Add(-1 * time.Hour),
		LastFullSyncTime: now.Add(-7*24*time.Hour - 1*time.Second), // 7 days + 1 second ago
		FullSyncInterval: 7 * 24 * time.Hour,
	}

	mode := decision.Decide(now)
	if mode != "full" {
		t.Errorf("7 days + 1 second should return 'full', got: %s", mode)
	}
}

func TestSyncModeDecision_LoadSyncTimeError(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	decision := SyncModeDecision{
		ForceFullSync:     false,
		LastSyncTime:      now.Add(-1 * time.Hour),
		LastFullSyncTime:  now.Add(-2 * 24 * time.Hour),
		FullSyncInterval:  7 * 24 * time.Hour,
		LoadSyncTimeError: &MockError{msg: "failed to load"},
	}

	mode := decision.Decide(now)
	if mode != "full" {
		t.Errorf("LoadSyncTimeError should return 'full', got: %s", mode)
	}
}

func TestSyncModeDecision_ZeroFullSyncTime(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	decision := SyncModeDecision{
		ForceFullSync:    false,
		LastSyncTime:     now.Add(-1 * time.Hour),
		LastFullSyncTime: time.Time{}, // Never had full sync
		FullSyncInterval: 7 * 24 * time.Hour,
	}

	mode := decision.Decide(now)
	// If LastFullSyncTime is zero, the condition !lastFullSyncTime.IsZero() is false
	// So the full sync check is skipped, result is incremental
	if mode != "incremental" {
		t.Errorf("Zero full sync time should return 'incremental', got: %s", mode)
	}
}

func TestSyncModeDecision_BoundaryConditions(t *testing.T) {
	testCases := []struct {
		name             string
		lastSyncHours    int // hours ago
		lastFullDays     int // days ago
		fullIntervalDays int
		expected         string
	}{
		{
			name:             "6 days ago - incremental",
			lastSyncHours:    1,
			lastFullDays:     6,
			fullIntervalDays: 7,
			expected:         "incremental",
		},
		{
			name:             "7 days ago - incremental (boundary)",
			lastSyncHours:    1,
			lastFullDays:     7,
			fullIntervalDays: 7,
			expected:         "incremental",
		},
		{
			name:             "8 days ago - full",
			lastSyncHours:    1,
			lastFullDays:     8,
			fullIntervalDays: 7,
			expected:         "full",
		},
		{
			name:             "30 days ago - full",
			lastSyncHours:    1,
			lastFullDays:     30,
			fullIntervalDays: 7,
			expected:         "full",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
			decision := SyncModeDecision{
				ForceFullSync:    false,
				LastSyncTime:     now.Add(-time.Duration(tc.lastSyncHours) * time.Hour),
				LastFullSyncTime: now.Add(-time.Duration(tc.lastFullDays) * 24 * time.Hour),
				FullSyncInterval: time.Duration(tc.fullIntervalDays) * 24 * time.Hour,
			}

			mode := decision.Decide(now)
			if mode != tc.expected {
				t.Errorf("Case %s: expected %s, got %s", tc.name, tc.expected, mode)
			}
		})
	}
}

type MockError struct {
	msg string
}

func (e *MockError) Error() string {
	return e.msg
}
