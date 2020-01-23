package testutil

import (
	"io"
	"math/rand"
)

// LoopbackReadWriter is a simple loopback pipe
type LoopbackReadWriter struct {
	io.Reader
	io.WriteCloser
}

// NewLoopback creates a LoopbackReadWriter
func NewLoopback() *LoopbackReadWriter {
	reader, writer := io.Pipe()

	return &LoopbackReadWriter{
		Reader:      reader,
		WriteCloser: writer,
	}
}

// RandomBytes returns a slice of random bytes
func RandomBytes(len int) []byte {
	result := make([]byte, len)

	for i := 0; i < len; i++ {
		result[i] = byte(rand.Uint32())
	}

	return result
}
