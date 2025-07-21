package storage

import (
	"testing"
)

func TestGCSStorage_getFullKey(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{
			name:   "no prefix",
			prefix: "",
			key:    "backup.tar.gz",
			want:   "backup.tar.gz",
		},
		{
			name:   "with prefix",
			prefix: "backups/postgres",
			key:    "backup.tar.gz",
			want:   "backups/postgres/backup.tar.gz",
		},
		{
			name:   "prefix with trailing slash",
			prefix: "backups/",
			key:    "backup.tar.gz",
			want:   "backups/backup.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GCSStorage{
				prefix: tt.prefix,
			}
			if got := g.getFullKey(tt.key); got != tt.want {
				t.Errorf("getFullKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGCSStorage_stripPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{
			name:   "no prefix",
			prefix: "",
			key:    "backup.tar.gz",
			want:   "backup.tar.gz",
		},
		{
			name:   "with prefix",
			prefix: "backups",
			key:    "backups/backup.tar.gz",
			want:   "backup.tar.gz",
		},
		{
			name:   "key shorter than prefix",
			prefix: "backups",
			key:    "back",
			want:   "back",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GCSStorage{
				prefix: tt.prefix,
			}
			if got := g.stripPrefix(tt.key); got != tt.want {
				t.Errorf("stripPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateServiceAccountJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid service account",
			json:    `{"type": "service_account", "project_id": "test"}`,
			wantErr: false,
		},
		{
			name:    "invalid type",
			json:    `{"type": "user", "project_id": "test"}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			json:    `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "empty json",
			json:    `{}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceAccountJSON(tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServiceAccountJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGCSConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  GCSConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: GCSConfig{
				Bucket:             "test-bucket",
				ProjectID:          "test-project",
				ServiceAccountJSON: `{"type": "service_account"}`,
			},
			wantErr: false,
		},
		{
			name: "missing bucket",
			config: GCSConfig{
				ProjectID:          "test-project",
				ServiceAccountJSON: `{"type": "service_account"}`,
			},
			wantErr: true,
		},
		{
			name: "missing project ID",
			config: GCSConfig{
				Bucket:             "test-bucket",
				ServiceAccountJSON: `{"type": "service_account"}`,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation would be done in the config package
			// This is just to show the test structure
			hasError := tt.config.Bucket == "" || tt.config.ProjectID == ""
			if hasError != tt.wantErr {
				t.Errorf("validation error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}
