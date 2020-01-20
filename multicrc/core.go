package multicrc

import (
	"sync"
)

//Params describes the parameters of a CRC. It also contains a table that is calculated on first use,
//therefore it is best to share the params as much as possible
type Params struct {
	Len           uint
	Name          string
	Polynomial    uint64
	ReflectInput  bool
	ReflectOutput bool
	InitialValue  uint64
	FinalXOR      uint64

	tableLock sync.Mutex
	table     interface{}
}

func makeMask(len uint) uint64 {
	return (uint64(1)<<(len) - 1)
}

func polynomialDivision(polynomial uint64, input uint64, len uint) uint64 {
	mask := makeMask(len)

	for i := uint(0); i < len; i++ {
		bitOut := input>>(len-1) > 0
		input <<= 1
		if bitOut {
			input ^= polynomial
		}
		input &= mask
	}

	return input
}

func (params *Params) makeTable() {
	params.tableLock.Lock()
	defer params.tableLock.Unlock()
	if params.table != nil {
		return
	}

	/* Sadly, no generics :( */
	if params.Len == 0 {
	} else if params.Len <= 8 {
		table := make([]uint8, 256)
		for i := 0; i < 256; i++ {
			table[i] = uint8(polynomialDivision(params.Polynomial, uint64(i), params.Len))
		}
		params.table = table
	} else if params.Len <= 16 {
		table := make([]uint16, 256)
		for i := 0; i < 256; i++ {
			table[i] = uint16(polynomialDivision(params.Polynomial, uint64(i), params.Len))
		}
		params.table = table
	} else if params.Len <= 16 {
		table := make([]uint32, 256)
		for i := 0; i < 256; i++ {
			table[i] = uint32(polynomialDivision(params.Polynomial, uint64(i), params.Len))
		}
		params.table = table
	} else if params.Len <= 64 {
		table := make([]uint64, 256)
		for i := 0; i < 256; i++ {
			table[i] = uint64(polynomialDivision(params.Polynomial, uint64(i), params.Len))
		}
		params.table = table
	} else {
		panic("CRC length too long")
	}
}

func (params *Params) updateCRC(shiftReg uint64, input []uint8) uint64 {
	if params.Len == 0 {
		return 0
	}

	for _, m := range input {
		if params.ReflectInput {
			m = reflectByte(m)
		}

		var tableIndex uint8
		if params.Len >= 8 {
			tableIndex = uint8(shiftReg>>(params.Len-8)) ^ m
		} else {
			tableIndex = uint8(shiftReg<<(8-params.Len)) ^ m
		}

		switch t := params.table.(type) {
		case []uint8:
			shiftReg = uint64(t[tableIndex])
		case []uint16:
			shiftReg = (shiftReg << 8) ^ uint64(t[tableIndex])
		case []uint32:
			shiftReg = (shiftReg << 8) ^ uint64(t[tableIndex])
		case []uint64:
			shiftReg = (shiftReg << 8) ^ t[tableIndex]
		default:
			panic("Wrong type of table")
		}
	}

	mask := makeMask(params.Len)

	return shiftReg & mask
}

func (params *Params) finalizeCRC(shiftReg uint64) uint64 {
	if params.Len == 0 {
		return 0
	}

	if params.ReflectOutput {
		shiftReg = reflectWithLen(shiftReg, params.Len)
	}

	return shiftReg ^ params.FinalXOR
}
