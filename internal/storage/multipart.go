// Package storage provides storage backend implementations.
package storage

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MultipartUploader handles multipart uploads for S3.
type MultipartUploader struct {
	client      *s3.Client
	bucket      string
	key         string
	uploadID    string
	parts       []types.CompletedPart
	partNumber  int32
	mu          sync.Mutex
	minPartSize int64
}

// NewMultipartUploader creates a new multipart uploader.
func NewMultipartUploader(client *s3.Client, bucket, key string) *MultipartUploader {
	return &MultipartUploader{
		client:      client,
		bucket:      bucket,
		key:         key,
		parts:       make([]types.CompletedPart, 0),
		partNumber:  1,
		minPartSize: 5 * 1024 * 1024, // 5MB minimum part size
	}
}

// Start initiates a multipart upload.
func (m *MultipartUploader) Start(ctx context.Context, metadata map[string]string) error {
	input := &s3.CreateMultipartUploadInput{
		Bucket:   aws.String(m.bucket),
		Key:      aws.String(m.key),
		Metadata: metadata,
	}

	output, err := m.client.CreateMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %w", err)
	}

	m.uploadID = *output.UploadId
	return nil
}

// UploadPart uploads a single part.
func (m *MultipartUploader) UploadPart(ctx context.Context, reader io.Reader, size int64) error {
	m.mu.Lock()
	partNumber := m.partNumber
	m.partNumber++
	m.mu.Unlock()

	input := &s3.UploadPartInput{
		Bucket:        aws.String(m.bucket),
		Key:           aws.String(m.key),
		UploadId:      aws.String(m.uploadID),
		PartNumber:    aws.Int32(partNumber),
		Body:          reader,
		ContentLength: aws.Int64(size),
	}

	output, err := m.client.UploadPart(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload part %d: %w", partNumber, err)
	}

	m.mu.Lock()
	m.parts = append(m.parts, types.CompletedPart{
		ETag:       output.ETag,
		PartNumber: aws.Int32(partNumber),
	})
	m.mu.Unlock()

	return nil
}

// Complete finalizes the multipart upload.
func (m *MultipartUploader) Complete(ctx context.Context) error {
	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(m.bucket),
		Key:      aws.String(m.key),
		UploadId: aws.String(m.uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: m.parts,
		},
	}

	_, err := m.client.CompleteMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// Abort cancels the multipart upload.
func (m *MultipartUploader) Abort(ctx context.Context) error {
	if m.uploadID == "" {
		return nil
	}

	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(m.bucket),
		Key:      aws.String(m.key),
		UploadId: aws.String(m.uploadID),
	}

	_, err := m.client.AbortMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	return nil
}

// StreamingMultipartUpload handles streaming multipart uploads.
func StreamingMultipartUpload(ctx context.Context, client *s3.Client, bucket, key string, reader io.Reader, metadata map[string]string) error {
	uploader := NewMultipartUploader(client, bucket, key)

	// Start multipart upload
	if err := uploader.Start(ctx, metadata); err != nil {
		return err
	}

	// Ensure cleanup on error
	var uploadErr error
	defer func() {
		if uploadErr != nil {
			// Try to abort the upload
			_ = uploader.Abort(context.Background())
		}
	}()

	// Buffer for reading parts
	buffer := make([]byte, uploader.minPartSize)

	for {
		n, err := io.ReadFull(reader, buffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			uploadErr = fmt.Errorf("failed to read data: %w", err)
			return uploadErr
		}

		if n > 0 {
			// Upload the part
			partReader := io.LimitReader(reader, int64(n))
			if uploadErr = uploader.UploadPart(ctx, partReader, int64(n)); uploadErr != nil {
				return uploadErr
			}
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
	}

	// Complete the upload
	if uploadErr = uploader.Complete(ctx); uploadErr != nil {
		return uploadErr
	}

	return nil
}
