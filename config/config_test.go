package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ATSEMAperiod != 5 {
		t.Fatalf("expected default ATSEMAperiod=5, got %d", cfg.ATSEMAperiod)
	}
	// Call Validate on a pointer to the config value.
	if err := (&cfg).Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestConfigValidation(t *testing.T) {
	cases := []struct {
		name    string
		modify  func(*IndicatorConfig)
		wantErr bool
	}{
		{
			name: "negative EMA period",
			modify: func(c *IndicatorConfig) {
				c.ATSEMAperiod = -1
			},
			wantErr: true,
		},
		{
			name: "zero EMA period",
			modify: func(c *IndicatorConfig) {
				c.ATSEMAperiod = 0
			},
			wantErr: true,
		},
		{
			name: "valid custom period",
			modify: func(c *IndicatorConfig) {
				c.ATSEMAperiod = 10
			},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		cfg := DefaultConfig()
		tc.modify(&cfg)
		// Again, invoke Validate on a pointer.
		err := (&cfg).Validate()
		if tc.wantErr && err == nil {
			t.Errorf("%s: expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		}
	}
}
