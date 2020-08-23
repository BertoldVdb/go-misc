package bufferfifo

import (
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	pdu "github.com/BertoldVdb/go-misc/pdubuf"
)

func writer(fifo *FIFO) {
	count := uint32(0)
	for {
		time.Sleep(time.Duration(rand.Float32()*200) * time.Microsecond)
		buf := pdu.Alloc(0, 4, 4)
		binary.LittleEndian.PutUint32(buf.Buf(), count)
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

		if binary.LittleEndian.Uint32(buf.Buf()) != count {
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

func TestClear(t *testing.T) {
	fifo := New(16)
	fifo.Push(pdu.Alloc(0, 1, 1))
	fifo.Push(pdu.Alloc(0, 1, 1))
	fifo.Push(pdu.Alloc(0, 1, 1))

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

	fifo.Push(pdu.Alloc(0, 1, 1))

	if fifo.Len() != 1 {
		t.Error("Wrong length returned after clear and insert")
	}
}
