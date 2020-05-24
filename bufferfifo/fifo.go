package bufferfifo

import "sync"

// FIFO is a simple queue designed to queue packets. It can also be used as a freelist using the
// PopOrCreate function
type FIFO struct {
	sync.Mutex

	ring [][]byte

	readPointer  int
	writePointer int
	elements     int

	allocSize int
}

// New creates the FIFO. All allocations will be a multiple of allocSize to reduce the need to reallocate
func New(allocSize int) *FIFO {
	if allocSize < 1 {
		allocSize = 1
	}

	return &FIFO{
		ring:      make([][]byte, allocSize),
		allocSize: allocSize,
	}
}

func (b *FIFO) incrementPointer(ptr *int) {
	*ptr++
	if *ptr >= len(b.ring) {
		*ptr = 0
	}
}

func (b *FIFO) popInternal() []byte {
	if b.elements == 0 {
		/* Empty... */
		return nil
	}

	e := b.ring[b.readPointer]
	b.ring[b.readPointer] = nil
	b.incrementPointer(&b.readPointer)
	b.elements--

	return e
}

// Pop removes and returns the first element of the FIFO. It there is no element nil is returned
func (b *FIFO) Pop() []byte {
	b.Lock()
	defer b.Unlock()

	return b.popInternal()
}

func (b *FIFO) newBuf(minLength int) []byte {
	n := ((minLength / b.allocSize) + 1) * b.allocSize
	return make([]byte, n)
}

// PopOrCreate delivers a buffer with the requested length. It may take it from the FIFO
// but may also allocate a new one if this is not possible.
func (b *FIFO) PopOrCreate(length int) []byte {
	buf := b.Pop()

	if cap(buf) < length {
		buf = b.newBuf(length)
	}

	buf = buf[:length]

	return buf
}

func (b *FIFO) reallocateInternal() {
	n := ((b.elements * 2 / b.allocSize) + 1) * b.allocSize
	if n == len(b.ring) {
		return
	}

	newRing := make([][]byte, n)

	wrIndex := 0
	for {
		e := b.popInternal()
		if e == nil {
			break
		}

		newRing[wrIndex] = e
		wrIndex++
	}

	b.readPointer = 0
	b.writePointer = wrIndex
	b.elements = wrIndex
	b.ring = newRing
}

// Reallocate reallocates the internal ring buffer. This can be used to free some memory after an episode of heavy load.
func (b *FIFO) Reallocate() {
	b.Lock()
	defer b.Unlock()

	b.reallocateInternal()
}

// Push inserts buf at the end of the FIFO. Returns number of elements in FIFO.
func (b *FIFO) Push(buf []byte) int {
	assert(buf != nil, "Cannot queue nil buffers")

	b.Lock()
	defer b.Unlock()

	if b.elements == len(b.ring) {
		/* Full, double size */
		b.reallocateInternal()
	}

	assert(b.ring[b.writePointer] == nil, "Ring at write pointer contained element!")
	b.ring[b.writePointer] = buf
	b.incrementPointer(&b.writePointer)
	b.elements++

	return b.elements
}

// Len returns the number of elements in the FIFO
func (b *FIFO) Len() int {
	b.Lock()
	defer b.Unlock()

	return b.elements
}

// Clear removes all elements form the FIFO and returns how many there were
func (b *FIFO) Clear() int {
	b.Lock()
	defer b.Unlock()

	i := 0
	for ; b.popInternal() != nil; i++ {
	}

	return i
}

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}
