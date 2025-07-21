// Package utils provides utility functions for the backup service.
package utils

import (
	"sync"
)

// BufferPool provides a pool of reusable byte buffers.
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a new buffer pool with buffers of the specified size.
func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		size: bufferSize,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
	}
}

// Get retrieves a buffer from the pool.
func (p *BufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put returns a buffer to the pool.
func (p *BufferPool) Put(buf []byte) {
	// Only return buffers of the expected size
	if cap(buf) == p.size {
		p.pool.Put(buf[:p.size])
	}
}

// DefaultBufferPool is a default buffer pool with 32KB buffers.
var DefaultBufferPool = NewBufferPool(32 * 1024)
