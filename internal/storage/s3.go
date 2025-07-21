package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"path"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage implements Storage interface for AWS S3.
type S3Storage struct {
	client       *s3.Client
	uploader     *manager.Uploader
	bucket       string
	prefix       string
	objectLock   bool
	usePathStyle bool
}

// S3Config holds S3-specific configuration.
type S3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
	Endpoint        string // Optional custom endpoint
	Prefix          string // Optional prefix for all keys
	ObjectLock      bool   // Enable object lock with MD5
	UsePathStyle    bool   // For S3-compatible services
}

// NewS3Storage creates a new S3 storage provider.
func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client options
	clientOpts := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = cfg.UsePathStyle
		},
	}

	// Add custom endpoint if provided
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, clientOpts...)

	// Create uploader
	uploader := manager.NewUploader(client)

	return &S3Storage{
		client:       client,
		uploader:     uploader,
		bucket:       cfg.Bucket,
		prefix:       cfg.Prefix,
		objectLock:   cfg.ObjectLock,
		usePathStyle: cfg.UsePathStyle,
	}, nil
}

// Upload implements Storage.Upload.
func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error {
	fullKey := s.getFullKey(key)

	input := &s3.PutObjectInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(fullKey),
		Body:     reader,
		Metadata: metadata,
	}

	// If object lock is enabled, calculate MD5
	if s.objectLock {
		data, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("failed to read data for MD5: %w", err)
		}

		// Calculate MD5
		hash := md5.Sum(data)
		contentMD5 := base64.StdEncoding.EncodeToString(hash[:])
		input.ContentMD5 = aws.String(contentMD5)

		// Reset reader with the data we read
		input.Body = bytes.NewReader(data)
	}

	// Upload the file
	_, err := s.uploader.Upload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// Delete implements Storage.Delete.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	fullKey := s.getFullKey(key)

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// List implements Storage.List.
func (s *S3Storage) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	fullPrefix := s.getFullKey(prefix)

	var objects []ObjectInfo
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(fullPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list S3 objects: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, ObjectInfo{
				Key:          s.stripPrefix(*obj.Key),
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
				Metadata:     make(map[string]string), // Metadata requires separate HEAD request
			})
		}
	}

	return objects, nil
}

// GetLastBackupTime implements Storage.GetLastBackupTime.
func (s *S3Storage) GetLastBackupTime(ctx context.Context) (time.Time, error) {
	objects, err := s.List(ctx, "")
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

	// Get metadata for the most recent object
	fullKey := s.getFullKey(objects[0].Key)
	headResp, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		// If we can't get metadata, return the last modified time
		return objects[0].LastModified, nil
	}

	// Check for backup timestamp in metadata
	if timestamp, ok := headResp.Metadata["backup-timestamp"]; ok {
		t, err := time.Parse(time.RFC3339, timestamp)
		if err == nil {
			return t, nil
		}
	}

	return objects[0].LastModified, nil
}

// getFullKey returns the full S3 key with prefix.
func (s *S3Storage) getFullKey(key string) string {
	if s.prefix == "" {
		return key
	}
	return path.Join(s.prefix, key)
}

// stripPrefix removes the storage prefix from a key.
func (s *S3Storage) stripPrefix(key string) string {
	if s.prefix == "" {
		return key
	}
	return key[len(s.prefix)+1:]
}

// readerAt wraps a byte slice to implement io.ReaderAt.
type readerAt struct {
	data []byte
}

func (r *readerAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n = copy(p, r.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}
