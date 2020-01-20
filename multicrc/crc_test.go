package multicrc

import (
	"bytes"
	"testing"
)

func TestCRCs(t *testing.T) {
	crc := NewCRC(&Params{
		Len:           16,
		Polynomial:    0x1021,
		InitialValue:  0xB2AA,
		ReflectOutput: true,
		ReflectInput:  true,
		FinalXOR:      0x0001,
	})

	crc.Reset().AddBytes([]byte{1, 2}).AddBytes([]byte{3})

	if crc.Result16() != 0x0fb0 {
		t.Error("Result is wrong")
	}

	var output [2]byte
	be := crc.ResultBytes(output[:], true)
	if !bytes.Equal(be, []byte{0xf, 0xb0}) {
		t.Error("Big endian result is wrong")
	}

	le := crc.ResultBytes(output[:], false)
	if !bytes.Equal(le, []byte{0xb0, 0xf}) {
		t.Error("Little endian result is wrong")
	}
}
