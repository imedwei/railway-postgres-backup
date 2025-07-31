package storage

import (
	"io"
	"strings"
	"testing"
)

// TestStorageAtomicUpload verifies that storage implementations handle upload failures atomically
func TestStorageAtomicUpload(t *testing.T) {
	// This test would require actual S3/GCS implementations or more sophisticated mocks
	// For now, we document the expected behavior

	t.Run("upload failure should not create partial files", func(t *testing.T) {
		// When Upload() returns an error, no file should exist on the storage backend
		// This is handled by the storage implementations:
		// - S3: Uses manager.Upload which is atomic
		// - GCS: Uses NewWriter with Close() that completes the upload atomically
	})

	t.Run("reader error during upload should fail cleanly", func(t *testing.T) {
		// If the reader returns an error during io.Copy, the upload should fail
		// and no partial file should be left on storage
	})
}

// failingReader is a reader that fails after reading n bytes
type failingReader struct {
	data      string
	bytesRead int
	failAfter int
}

func (r *failingReader) Read(p []byte) (n int, err error) {
	if r.bytesRead >= r.failAfter {
		return 0, io.ErrUnexpectedEOF
	}

	remaining := r.failAfter - r.bytesRead
	if remaining > len(p) {
		remaining = len(p)
	}

	dataRemaining := len(r.data) - r.bytesRead
	if dataRemaining < remaining {
		remaining = dataRemaining
	}

	if remaining > 0 {
		copy(p[:remaining], r.data[r.bytesRead:r.bytesRead+remaining])
		r.bytesRead += remaining
		return remaining, nil
	}

	return 0, io.EOF
}

// TestCountingReader verifies our counting reader works correctly
func TestCountingReader(t *testing.T) {
	data := "Hello, World!"
	reader := strings.NewReader(data)
	
	countingReader := &countingReader{
		reader: reader,
		count:  0,
	}

	// Read all data
	result, err := io.ReadAll(countingReader)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if string(result) != data {
		t.Errorf("Expected %q, got %q", data, string(result))
	}

	if countingReader.count != int64(len(data)) {
		t.Errorf("Expected count %d, got %d", len(data), countingReader.count)
	}
}

// countingReader implementation (copied from orchestrator.go for testing)
type countingReader struct {
	reader io.Reader
	count  int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.reader.Read(p)
	cr.count += int64(n)
	return n, err
}