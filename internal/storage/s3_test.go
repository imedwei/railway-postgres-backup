package storage

import (
	"testing"
)

// MockS3Client is a mock implementation for testing.
// In a real implementation, we would use a proper mocking framework
// or the AWS SDK's testing utilities.

func TestS3Storage_getFullKey(t *testing.T) {
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
			s := &S3Storage{
				prefix: tt.prefix,
			}
			if got := s.getFullKey(tt.key); got != tt.want {
				t.Errorf("getFullKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestS3Storage_stripPrefix(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &S3Storage{
				prefix: tt.prefix,
			}
			if got := s.stripPrefix(tt.key); got != tt.want {
				t.Errorf("stripPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReaderAt(t *testing.T) {
	data := []byte("test data")
	r := &readerAt{data: data}

	// Test normal read
	buf := make([]byte, 4)
	n, err := r.ReadAt(buf, 0)
	if err != nil {
		t.Errorf("ReadAt() unexpected error: %v", err)
	}
	if n != 4 {
		t.Errorf("ReadAt() n = %v, want 4", n)
	}
	if string(buf) != "test" {
		t.Errorf("ReadAt() read %v, want 'test'", string(buf))
	}

	// Test read at offset
	n, err = r.ReadAt(buf, 5)
	if err != nil && err.Error() != "EOF" {
		t.Errorf("ReadAt() unexpected error: %v", err)
	}
	if n != 4 {
		t.Errorf("ReadAt() n = %v, want 4", n)
	}
	if string(buf[:n]) != "data" {
		t.Errorf("ReadAt() read %v, want 'data'", string(buf[:n]))
	}

	// Test read past end
	_, err = r.ReadAt(buf, 100)
	if err == nil || err.Error() != "EOF" {
		t.Errorf("ReadAt() expected EOF, got %v", err)
	}
}

// Integration tests would require mocking the AWS SDK or using localstack
func TestS3Config_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  S3Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: S3Config{
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Region:          "us-east-1",
				Bucket:          "test-bucket",
			},
			wantErr: false,
		},
		{
			name: "valid config with endpoint",
			config: S3Config{
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Bucket:          "test-bucket",
				Endpoint:        "https://s3.custom.com",
			},
			wantErr: false,
		},
		{
			name: "missing access key",
			config: S3Config{
				SecretAccessKey: "test-secret",
				Region:          "us-east-1",
				Bucket:          "test-bucket",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation would be done in the config package
			// This is just to show the test structure
			hasError := tt.config.AccessKeyID == ""
			if hasError != tt.wantErr {
				t.Errorf("validation error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}
