package bufferedpipe

import (
	"bytes"
	"errors"
    "io"
    "sync"
)

// BufferedPipe is an io.ReadWriteCloser. What is written via the writer comes out via the Reader.
// A configurable buffer is present in between. Writing behaviour can be configured both in blocking
// and non blocking ways.
type BufferedPipe struct {
    io.ReadWriteCloser
    sync.Mutex
	buffer bytes.Buffer

	canReadSignal  chan (struct{})
	canWriteSignal chan (struct{})

	maximumCapacity    int
	WriteAllowTruncate bool
	WriteBlocks        bool

	closed bool
}

var (
	// ErrorWriteTruncated is returned when a non-blocking write does not fit in the buffer
	// and was truncated
	ErrorWriteTruncated = errors.New("Write truncated due to full write buffer")

	// ErrorWriteFull is returned when a non-blocking write does not fit in the buffer
	// and was not performed
	ErrorWriteFull = errors.New("Write ignored due to full write buffer")

	// ErrorClosed is returned when the caller tries to write to a closed pipe, or tries to read
	// from a closed and empty pipe
	ErrorClosed = errors.New("The pipe is closed")
)

func signalChannel(c chan (struct{})) {
	select {
	case c <- struct{}{}:
	default:
	}
}

func (b *BufferedPipe) remainingCapacity() int {
    if b.maximumCapacity <= 0 {
        return 0;
    }

	result := b.maximumCapacity - b.buffer.Len()
	assert(result >= 0, "Maximum capacity exceeded")
	return result
}

// MaximumCapacity returns the maximum amount of bytes that can be stored in the pipe
func (b *BufferedPipe) MaximumCapacity() int {
	return b.maximumCapacity
}

// RemainingCapacity returns the minimum amount of bytes that can be written to the pipe
// without blocking for truncation
func (b *BufferedPipe) RemainingCapacity() int {
	b.Lock()
	defer b.Unlock()

	return b.remainingCapacity()
}

// Len returns the number of bytes currently stored in the pipe
func (b *BufferedPipe) Len() int {
	b.Lock()
	defer b.Unlock()

	return b.buffer.Len()
}

// Clear clears the internal buffer.
func (b *BufferedPipe) Clear() {
	b.Lock()
	defer b.Unlock()

	b.buffer.Reset()
}

// Close closes the pipe. Read calls will return ErrorClosed when the pipe is exhausted.
// Write call will return ErrorClosed right away
func (b *BufferedPipe) Close() error {
	b.Lock()
	defer b.Unlock()

	b.closed = true

	/* Unblock waiting goroutines */
	signalChannel(b.canReadSignal)
	signalChannel(b.canWriteSignal)

    return nil
}

func (b *BufferedPipe) writeNonBlocking(p []byte) (int, error) {
	writeBuf := p
	truncated := false

	b.Lock()
	if b.closed {
		return 0, ErrorClosed
	}

	if b.maximumCapacity > 0 {
		remainingCapacity := b.remainingCapacity()

		if len(writeBuf) > remainingCapacity {
			if !b.WriteAllowTruncate {
				b.Unlock()
				return 0, ErrorWriteFull
			}

			writeBuf = writeBuf[:remainingCapacity]
			truncated = true
		}
	}

	n, err := b.buffer.Write(writeBuf)
	b.Unlock()

	signalChannel(b.canReadSignal)

	if err == nil && truncated {
		err = ErrorWriteTruncated
	}

	return n, err
}

func (b *BufferedPipe) writeBlocking(p []byte) (int, error) {
	doWrite := func(k []byte) (int, error) {
		n, err := b.buffer.Write(k)

		signalChannel(b.canReadSignal)

		if b.remainingCapacity() > 0 {
			/* Another goroutine can potentially also write */
			signalChannel(b.canWriteSignal)
		}

		return n, err
	}

	totalWritten := 0

	for {
		b.Lock()
		if b.closed {
			signalChannel(b.canWriteSignal)
			b.Unlock()
			return totalWritten, ErrorClosed
		}

		remainingCapacity := b.remainingCapacity()

		if b.maximumCapacity <= 0 || remainingCapacity >= len(p) {
			n, err := doWrite(p)
			totalWritten += n
			b.Unlock()
			return totalWritten, err
		}

		if remainingCapacity > 0 {
			n, err := doWrite(p[:remainingCapacity])
			p = p[remainingCapacity:]
			totalWritten += n

			assert(err == nil, "bytes.Buffer write returned err != nil. This is not allowed by the documentation")
		}

		b.Unlock()

		<-b.canWriteSignal
	}
}

// Write implements the write function of io.Writer
func (b *BufferedPipe) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if !b.WriteBlocks {
		return b.writeNonBlocking(p)
	}

	return b.writeBlocking(p)
}

// Read implements the read function of io.Reader
func (b *BufferedPipe) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	for {
		b.Lock()
		n, err := b.buffer.Read(p)

		if n > 0 {
			if b.buffer.Len() > 0 {
				/* Another goroutine can potentially also read */
				signalChannel(b.canReadSignal)
			}

			signalChannel(b.canWriteSignal)
			b.Unlock()

			return n, err
		}

		if b.closed {
			signalChannel(b.canReadSignal)
			b.Unlock()
			return 0, ErrorClosed
		}

		b.Unlock()

		<-b.canReadSignal
	}
}

// NewBufferedPipe constructs a new pipe with the stated maximum capacity. If maximumCapacity is zero or less, the
// capacity is not bounded.
func NewBufferedPipe(maximumCapacity int) *BufferedPipe {
	return &BufferedPipe{
		maximumCapacity:    maximumCapacity,
		canReadSignal:      make(chan (struct{}), 1),
		canWriteSignal:     make(chan (struct{}), 1),
		WriteAllowTruncate: false,
		WriteBlocks:        true,
	}
}

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}
