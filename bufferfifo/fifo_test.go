package bufferfifo

import (
	"encoding/binary"
	"math/rand"
	"testing"
	"time"
)

func writer(fifo *FIFO) {
	count := uint32(0)
	for {
		time.Sleep(time.Duration(rand.Float32()*200) * time.Microsecond)
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, count)
		fifo.Push(buf)
		count++

		if count == 5000 {
			return
		}
	}
}

func reader(t *testing.T, fifo *FIFO) {
	count := uint32(0)
	for {
		time.Sleep(time.Duration(rand.Float32()*200) * time.Microsecond)
		buf := fifo.Pop()
		if buf == nil {
			continue
		}

		if binary.LittleEndian.Uint32(buf) != count {
			t.Error("Wrong element returned in pop")
		}
		count++

		if count%1000 == 0 {
			/* Ensure the fifo needs to grow */
			time.Sleep(20 * time.Millisecond)
		}

		if count == 1600 {
			/* Do a reallocation */
			fifo.Reallocate()
			fifo.Reallocate()
		}

		if count == 5000 {
			return
		}
	}
}

func TestBasic(t *testing.T) {
	fifo := New(0)

	go writer(fifo)

	reader(t, fifo)
}

func TestAssert(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	assert(false, "Assert failed")
}

func TestCreate(t *testing.T) {
	fifo := New(16)
	buf1 := fifo.PopOrCreate(12)
	if len(buf1) != 12 || cap(buf1) != 16 {
		t.Error("Returned buffer with wrong size (1)")
	}

	buf2 := fifo.PopOrCreate(24)
	if len(buf2) != 24 || cap(buf2) != 32 {
		t.Error("Returned buffer with wrong size (2)")
	}

	fifo.Push(buf1[:2])
	fifo.Push(buf2[:2])

	buf3 := fifo.PopOrCreate(12)
	if len(buf3) != 12 || cap(buf3) != 16 {
		t.Error("Returned buffer with wrong size (3)")
	}

	buf4 := fifo.PopOrCreate(12)
	if len(buf4) != 12 || cap(buf4) != 32 {
		t.Error("Returned buffer with wrong size (4)")
	}

	if &buf1[0] != &buf3[0] || &buf2[0] != &buf4[0] {
		t.Error("New buffers were created and this wat not needed")
	}
}

func TestClear(t *testing.T) {
	fifo := New(16)
	fifo.Push(make([]byte, 1))
	fifo.Push(make([]byte, 1))
	fifo.Push(make([]byte, 1))

	if fifo.Len() != 3 {
		t.Error("Wrong length returned after 3 insert")
	}

	fifo.Pop()

	if fifo.Len() != 2 {
		t.Error("Wrong length returned after pop")
	}

	if fifo.Clear() != 2 {
		t.Error("Clear returned wrong length")
	}

	if fifo.Len() != 0 {
		t.Error("Wrong length returned after clear")
	}

	fifo.Push(make([]byte, 1))

	if fifo.Len() != 1 {
		t.Error("Wrong length returned after clear and insert")
	}
}
