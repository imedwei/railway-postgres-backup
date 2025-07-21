package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCSStorage implements Storage interface for Google Cloud Storage.
type GCSStorage struct {
	client *storage.Client
	bucket string
	prefix string
}

// GCSConfig holds GCS-specific configuration.
type GCSConfig struct {
	Bucket             string
	ProjectID          string
	ServiceAccountJSON string
	Prefix             string // Optional prefix for all keys
	CustomerManagedKey string // Optional CMEK
}

// NewGCSStorage creates a new GCS storage provider.
func NewGCSStorage(ctx context.Context, cfg GCSConfig) (*GCSStorage, error) {
	// Parse service account JSON
	var opts []option.ClientOption
	if cfg.ServiceAccountJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(cfg.ServiceAccountJSON)))
	}

	// Create GCS client
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCSStorage{
		client: client,
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
	}, nil
}

// Upload implements Storage.Upload.
func (g *GCSStorage) Upload(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error {
	fullKey := g.getFullKey(key)

	// Get bucket handle
	bucket := g.client.Bucket(g.bucket)
	obj := bucket.Object(fullKey)

	// Create writer
	w := obj.NewWriter(ctx)
	w.Metadata = metadata

	// Copy data
	if _, err := io.Copy(w, reader); err != nil {
		_ = w.Close()
		return fmt.Errorf("failed to upload to GCS: %w", err)
	}

	// Close writer to complete upload
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to finalize GCS upload: %w", err)
	}

	return nil
}

// Delete implements Storage.Delete.
func (g *GCSStorage) Delete(ctx context.Context, key string) error {
	fullKey := g.getFullKey(key)

	bucket := g.client.Bucket(g.bucket)
	obj := bucket.Object(fullKey)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete from GCS: %w", err)
	}

	return nil
}

// List implements Storage.List.
func (g *GCSStorage) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	fullPrefix := g.getFullKey(prefix)

	var objects []ObjectInfo
	bucket := g.client.Bucket(g.bucket)

	query := &storage.Query{
		Prefix: fullPrefix,
	}

	it := bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list GCS objects: %w", err)
		}

		objects = append(objects, ObjectInfo{
			Key:          g.stripPrefix(attrs.Name),
			Size:         attrs.Size,
			LastModified: attrs.Updated,
			Metadata:     attrs.Metadata,
		})
	}

	return objects, nil
}

// GetLastBackupTime implements Storage.GetLastBackupTime.
func (g *GCSStorage) GetLastBackupTime(ctx context.Context) (time.Time, error) {
	objects, err := g.List(ctx, "")
	if err != nil {
		return time.Time{}, err
	}

	if len(objects) == 0 {
		return time.Time{}, nil
	}

	// Sort by last modified time descending
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.After(objects[j].LastModified)
	})

	// Check for backup timestamp in metadata
	if timestamp, ok := objects[0].Metadata["backup-timestamp"]; ok {
		t, err := time.Parse(time.RFC3339, timestamp)
		if err == nil {
			return t, nil
		}
	}

	return objects[0].LastModified, nil
}

// Close closes the GCS client connection.
func (g *GCSStorage) Close() error {
	return g.client.Close()
}

// getFullKey returns the full GCS object name with prefix.
func (g *GCSStorage) getFullKey(key string) string {
	if g.prefix == "" {
		return key
	}
	return path.Join(g.prefix, key)
}

// stripPrefix removes the storage prefix from a key.
func (g *GCSStorage) stripPrefix(key string) string {
	if g.prefix == "" {
		return key
	}
	if len(key) > len(g.prefix) {
		return key[len(g.prefix)+1:]
	}
	return key
}

// ValidateServiceAccountJSON validates the service account JSON string.
func ValidateServiceAccountJSON(jsonStr string) error {
	var sa struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &sa); err != nil {
		return fmt.Errorf("invalid service account JSON: %w", err)
	}

	if sa.Type != "service_account" {
		return fmt.Errorf("invalid service account type: %s", sa.Type)
	}

	return nil
}
