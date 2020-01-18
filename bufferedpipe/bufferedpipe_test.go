package bufferedpipe

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func writeUntilError(p io.Writer, data []byte) (int, error) {
	count := 0
	for {
		n, err := p.Write(data)
		count += n
		if err != nil {
			return count, err
		}
	}
}

func testBuffer(in []byte, check []byte) bool {
	for len(in) > 0 {
		k := len(in)
		if k > len(check) {
			k = len(check)
		}

		if !bytes.Equal(in[:k], check[:k]) {
			return false
		}

		in = in[k:]
	}

	return true
}

func TestClear(t *testing.T) {
	b := NewBufferedPipe(100)
	b.Write([]byte{5, 5, 4, 4})
	if b.Len() != 4 {
		t.Error("Wrong length before clear")
	}
	b.Clear()
	if b.Len() != 0 {
		t.Error("Wrong length after clear")
	}
}

func TestNonBlocking(t *testing.T) {
	b := NewBufferedPipe(100)
	var readBuf [256]byte
	b.WriteBlocks = false

	testBuf := []byte{1, 2, 3}

	count, err := writeUntilError(b, testBuf)
	if err != ErrorWriteFull {
		t.Error("Wrong error in non-blocking non-truncating case")
	}

	n, err := b.Read(readBuf[:])
	if err != nil {
		t.Error("Got error when reading")
	}
	if count != 99 || n != 99 {
		t.Error("Wrong number of bytes written in non-blocking non-truncating case", count, n)
	}
	if b.Len() != 0 {
		t.Error("Still bytes left after reading it all")
	}
	if !testBuffer(readBuf[:n], testBuf) {
		t.Error("Read value was wrong")
	}

	b.WriteAllowTruncate = true
	b.Clear()

	count, err = writeUntilError(b, []byte{1, 2, 3})
	if err != ErrorWriteTruncated {
		t.Error("Wrong error in non-blocking truncating case")
	}
	n, err = b.Read(readBuf[:])
	if err != nil {
		t.Error("Got error when reading")
	}
	if count != 100 || n != 100 {
		t.Error("Wrong number of bytes written in non-blocking truncating case", count, n)
	}
	if b.Len() != 0 {
		t.Error("Still bytes left after reading it all")
	}
	if !testBuffer(readBuf[:n], testBuf) {
		t.Error("Read value was wrong")
	}

	b.Close()

	n, err = b.Write([]byte("test"))
	if err != ErrorClosed || n != 0 {
		t.Error("Writing to closed pipe did not return error")
	}
}

func testLen(t *testing.T, b *BufferedPipe) {
	max := b.MaximumCapacity()
	len := b.Len()
	cap := b.RemainingCapacity()

	if len > max || len < 0 {
		t.Error("Invalid length returned", len, max)
	}

	if cap > max || len < 0 {
		t.Error("Invalid capacity returned", len, max)
	}
}

func testBlocking(t *testing.T, b *BufferedPipe) {
	done := make(chan (uint64), 2)

	go func() {
		counter := byte(0)
		totalBytes := uint64(0)

		for {
			n, err := b.Write([]byte{counter, counter + 1, counter + 2})
			testLen(t, b)

			counter += byte(n)
			totalBytes += uint64(n)
			if err != nil {
				if err != ErrorClosed {
					t.Error("Got unexpected error on write", err, n)
				}

				done <- totalBytes
				return
			}
		}
	}()

	go func() {
		counter := byte(0)
		totalBytes := uint64(0)

		var rxBuf [11]byte
		for {
			n, err := b.Read(rxBuf[:])
			testLen(t, b)

			totalBytes += uint64(n)
			if err != nil {
				if err != ErrorClosed {
					t.Error("Got unexpected error on read", err, n)
				}

				done <- totalBytes
				return
			}

			for _, m := range rxBuf[:n] {
				if m != counter {
					t.Error("Read wrong value", m, counter)
				}
				counter++
			}
		}
	}()

	time.Sleep(5 * time.Second)
	b.Close()

	n1 := <-done
	n2 := <-done
	if n1 != n2 {
		t.Error("Different number of bytes written versus bytes read")
	}
}

func TestBlocking(t *testing.T) {
	testBlocking(t, NewBufferedPipe(100))
	testBlocking(t, NewBufferedPipe(5))
	testBlocking(t, NewBufferedPipe(10000))
	testBlocking(t, NewBufferedPipe(1024*1024))
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

func TestEmptyReadWrite(t *testing.T) {
	b := NewBufferedPipe(100)

	n, err := b.Write(nil)
	if n != 0 || err != nil {
		t.Error("Nil write failed", n, err)
	}

	n, err = b.Write([]byte{})
	if n != 0 || err != nil {
		t.Error("Empty write failed", n, err)
	}

	n, err = b.Read(nil)
	if n != 0 || err != nil {
		t.Error("Nil read failed", n, err)
	}

	var buf []byte
	n, err = b.Read(buf)
	if n != 0 || err != nil {
		t.Error("Empty read failed", n, err)
	}
}
