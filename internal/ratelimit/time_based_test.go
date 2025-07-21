package ratelimit

import (
	"strings"
	"testing"
	"time"
)

func TestTimeBasedLimiter_ShouldBackup(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		lastBackup     time.Time
		wantAllow      bool
		wantReasonPart string
	}{
		{
			name: "no previous backup",
			config: Config{
				MinInterval: 6 * time.Hour,
				ForceBackup: false,
			},
			lastBackup:     time.Time{},
			wantAllow:      true,
			wantReasonPart: "no previous backup",
		},
		{
			name: "forced backup",
			config: Config{
				MinInterval: 6 * time.Hour,
				ForceBackup: true,
			},
			lastBackup:     time.Now().Add(-1 * time.Hour),
			wantAllow:      true,
			wantReasonPart: "forced backup",
		},
		{
			name: "backup too recent",
			config: Config{
				MinInterval: 6 * time.Hour,
				ForceBackup: false,
			},
			lastBackup:     time.Now().Add(-2 * time.Hour),
			wantAllow:      false,
			wantReasonPart: "next backup allowed in",
		},
		{
			name: "backup allowed after interval",
			config: Config{
				MinInterval: 6 * time.Hour,
				ForceBackup: false,
			},
			lastBackup:     time.Now().Add(-7 * time.Hour),
			wantAllow:      true,
			wantReasonPart: "last backup was",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewTimeBasedLimiter(tt.config)
			gotAllow, gotReason := limiter.ShouldBackup(tt.lastBackup)

			if gotAllow != tt.wantAllow {
				t.Errorf("ShouldBackup() gotAllow = %v, want %v", gotAllow, tt.wantAllow)
			}

			if !strings.Contains(gotReason, tt.wantReasonPart) {
				t.Errorf("ShouldBackup() gotReason = %v, want to contain %v", gotReason, tt.wantReasonPart)
			}
		})
	}
}

func TestTimeBasedLimiter_GetMinInterval(t *testing.T) {
	config := Config{
		MinInterval: 8 * time.Hour,
	}
	limiter := NewTimeBasedLimiter(config)

	if got := limiter.GetMinInterval(); got != config.MinInterval {
		t.Errorf("GetMinInterval() = %v, want %v", got, config.MinInterval)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Second, "30 seconds"},
		{90 * time.Second, "2 minutes"},
		{45 * time.Minute, "45 minutes"},
		{90 * time.Minute, "1.5 hours"},
		{25 * time.Hour, "25.0 hours"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatDuration(tt.duration); got != tt.want {
				t.Errorf("formatDuration(%v) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}
