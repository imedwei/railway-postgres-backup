// Package utils provides utility functions for the backup service.
package utils

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// ProgressReader wraps an io.Reader and tracks bytes read.
type ProgressReader struct {
	reader      io.Reader
	bytesRead   atomic.Int64
	startTime   time.Time
	lastUpdate  time.Time
	updateFunc  func(bytesRead int64, elapsed time.Duration)
	updateEvery int64
}

// NewProgressReader creates a new progress tracking reader.
func NewProgressReader(reader io.Reader, updateFunc func(bytesRead int64, elapsed time.Duration)) *ProgressReader {
	return &ProgressReader{
		reader:      reader,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		updateFunc:  updateFunc,
		updateEvery: 10 * 1024 * 1024, // Update every 10MB
	}
}

// Read implements io.Reader interface with progress tracking.
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	if n > 0 {
		newTotal := pr.bytesRead.Add(int64(n))

		// Check if we should update progress
		if pr.updateFunc != nil && (newTotal%pr.updateEvery) < int64(n) {
			elapsed := time.Since(pr.startTime)
			pr.updateFunc(newTotal, elapsed)
			pr.lastUpdate = time.Now()
		}
	}
	return n, err
}

// BytesRead returns the total number of bytes read.
func (pr *ProgressReader) BytesRead() int64 {
	return pr.bytesRead.Load()
}

// ProgressWriter wraps an io.Writer and tracks bytes written.
type ProgressWriter struct {
	writer       io.Writer
	bytesWritten atomic.Int64
	startTime    time.Time
	updateFunc   func(bytesWritten int64, elapsed time.Duration)
	updateEvery  int64
}

// NewProgressWriter creates a new progress tracking writer.
func NewProgressWriter(writer io.Writer, updateFunc func(bytesWritten int64, elapsed time.Duration)) *ProgressWriter {
	return &ProgressWriter{
		writer:      writer,
		startTime:   time.Now(),
		updateFunc:  updateFunc,
		updateEvery: 10 * 1024 * 1024, // Update every 10MB
	}
}

// Write implements io.Writer interface with progress tracking.
func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.writer.Write(p)
	if n > 0 {
		newTotal := pw.bytesWritten.Add(int64(n))

		// Check if we should update progress
		if pw.updateFunc != nil && (newTotal%pw.updateEvery) < int64(n) {
			elapsed := time.Since(pw.startTime)
			pw.updateFunc(newTotal, elapsed)
		}
	}
	return n, err
}

// BytesWritten returns the total number of bytes written.
func (pw *ProgressWriter) BytesWritten() int64 {
	return pw.bytesWritten.Load()
}

// FormatBytes formats bytes in human-readable format.
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatRate formats transfer rate in human-readable format.
func FormatRate(bytesPerSecond float64) string {
	return fmt.Sprintf("%s/s", FormatBytes(int64(bytesPerSecond)))
}
