package multicrc

func reflectByte(x uint8) uint8 {
	x = (x&0x0F)<<4 | (x&0xF0)>>4
	x = (x&0x33)<<2 | (x&0xCC)>>2
	x = (x&0x55)<<1 | (x&0xAA)>>1
	return x
}

func reflectWithLen(x uint64, len uint) uint64 {
	if len <= 8 {
		return uint64(reflectByte(uint8(x))) >> (8 - len)
	}

	result := uint64(0)
	for i := uint(0); i < len; i += 8 {
		result |= uint64(reflectByte(uint8(x>>i))) << (len - i - 8)
	}
	return result
}
