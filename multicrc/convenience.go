package multicrc

//Result8 converts the CRC result to uint8
func (c *CRC) Result8() uint8 {
	if c.params.Len > 8 {
		panic("Result does not fit in uint8")
	}
	return uint8(c.Result64())
}

//Result16 converts the CRC result to uint16
func (c *CRC) Result16() uint16 {
	if c.params.Len > 16 {
		panic("Result does not fit in uint16")
	}
	return uint16(c.Result64())
}

//Result32 converts the CRC result to uint32
func (c *CRC) Result32() uint32 {
	if c.params.Len > 32 {
		panic("Result does not fit in uint32")
	}
	return uint32(c.Result64())
}

//ResultBytes converts the CRC result to a byte array with specified endianness
func (c *CRC) ResultBytes(output []byte, bigEndian bool) []byte {
	if c.params.Len == 0 {
		return nil
	}

	output = output[:c.ResultLenBytes()]
	result := c.Result64()

	for i := 0; i < len(output); i++ {
		if bigEndian {
			output[len(output)-1-i] = byte(result)
		} else {
			output[i] = byte(result)
		}
		result >>= 8
	}

	return output
}

//ResultLenBytes returns the length of the CRC in bytes
func (c *CRC) ResultLenBytes() int {
	if c.params.Len == 0 {
		return 0
	}
	return int((c.params.Len-1)/8 + 1)
}
