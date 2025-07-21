package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original env
	originalEnv := map[string]string{
		"DATABASE_URL":             os.Getenv("DATABASE_URL"),
		"STORAGE_PROVIDER":         os.Getenv("STORAGE_PROVIDER"),
		"AWS_ACCESS_KEY_ID":        os.Getenv("AWS_ACCESS_KEY_ID"),
		"AWS_SECRET_ACCESS_KEY":    os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"S3_BUCKET":                os.Getenv("S3_BUCKET"),
		"S3_REGION":                os.Getenv("S3_REGION"),
		"RESPAWN_PROTECTION_HOURS": os.Getenv("RESPAWN_PROTECTION_HOURS"),
	}
	defer func() {
		for k, v := range originalEnv {
			os.Setenv(k, v)
		}
	}()

	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "valid S3 config",
			env: map[string]string{
				"DATABASE_URL":          "postgres://user:pass@localhost/db",
				"STORAGE_PROVIDER":      "s3",
				"AWS_ACCESS_KEY_ID":     "test-key",
				"AWS_SECRET_ACCESS_KEY": "test-secret",
				"S3_BUCKET":             "test-bucket",
				"S3_REGION":             "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "valid GCS config",
			env: map[string]string{
				"DATABASE_URL":                "postgres://user:pass@localhost/db",
				"STORAGE_PROVIDER":            "gcs",
				"GCS_BUCKET":                  "test-bucket",
				"GOOGLE_PROJECT_ID":           "test-project",
				"GOOGLE_SERVICE_ACCOUNT_JSON": `{"type": "service_account"}`,
			},
			wantErr: false,
		},
		{
			name: "missing DATABASE_URL",
			env: map[string]string{
				"STORAGE_PROVIDER": "s3",
			},
			wantErr: true,
		},
		{
			name: "missing STORAGE_PROVIDER",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost/db",
			},
			wantErr: true,
		},
		{
			name: "invalid STORAGE_PROVIDER",
			env: map[string]string{
				"DATABASE_URL":     "postgres://user:pass@localhost/db",
				"STORAGE_PROVIDER": "invalid",
			},
			wantErr: true,
		},
		{
			name: "S3 with custom endpoint",
			env: map[string]string{
				"DATABASE_URL":          "postgres://user:pass@localhost/db",
				"STORAGE_PROVIDER":      "s3",
				"AWS_ACCESS_KEY_ID":     "test-key",
				"AWS_SECRET_ACCESS_KEY": "test-secret",
				"S3_BUCKET":             "test-bucket",
				"S3_ENDPOINT":           "https://s3.custom.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env
			for k := range originalEnv {
				os.Unsetenv(k)
			}

			// Set test env
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cfg == nil {
				t.Errorf("Load() returned nil config without error")
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid S3 config",
			config: Config{
				DatabaseURL:        "postgres://localhost",
				StorageProvider:    "s3",
				AWSAccessKeyID:     "key",
				AWSSecretAccessKey: "secret",
				S3Bucket:           "bucket",
				S3Region:           "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "missing S3 credentials",
			config: Config{
				DatabaseURL:     "postgres://localhost",
				StorageProvider: "s3",
				S3Bucket:        "bucket",
				S3Region:        "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "negative respawn protection",
			config: Config{
				DatabaseURL:            "postgres://localhost",
				StorageProvider:        "s3",
				AWSAccessKeyID:         "key",
				AWSSecretAccessKey:     "secret",
				S3Bucket:               "bucket",
				S3Region:               "us-east-1",
				RespawnProtectionHours: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_GetRespawnProtectionDuration(t *testing.T) {
	cfg := &Config{
		RespawnProtectionHours: 8,
	}

	want := 8 * time.Hour
	if got := cfg.GetRespawnProtectionDuration(); got != want {
		t.Errorf("GetRespawnProtectionDuration() = %v, want %v", got, want)
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	if got := getEnvInt("TEST_INT", 10); got != 42 {
		t.Errorf("getEnvInt() = %v, want %v", got, 42)
	}

	if got := getEnvInt("TEST_INT_MISSING", 10); got != 10 {
		t.Errorf("getEnvInt() with missing key = %v, want %v", got, 10)
	}
}

func TestGetEnvBool(t *testing.T) {
	os.Setenv("TEST_BOOL", "true")
	defer os.Unsetenv("TEST_BOOL")

	if got := getEnvBool("TEST_BOOL", false); got != true {
		t.Errorf("getEnvBool() = %v, want %v", got, true)
	}

	if got := getEnvBool("TEST_BOOL_MISSING", true); got != true {
		t.Errorf("getEnvBool() with missing key = %v, want %v", got, true)
	}
}
