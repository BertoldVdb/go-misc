package multicrc

//The CRC struct is used to calculate a CRC
type CRC struct {
	params   *Params
	shiftReg uint64
}

//NewCRC creates a CRC object with the specified params. If the table
//is not yet calculated this method will do it. It will also call Reset()
//internally
func NewCRC(params *Params) *CRC {
	c := &CRC{
		params: params,
	}

	params.makeTable()

	return c.Reset()
}

//GetName returns the name of the used CRC type
func (c *CRC) GetName() string {
	return c.params.Name
}

//Reset resets the CRC object to prepare it for a new calculation.
func (c *CRC) Reset() *CRC {
	c.shiftReg = c.params.InitialValue
	return c
}

//AddBytes adds the specified bytes to the CRC calculation
func (c *CRC) AddBytes(input []byte) *CRC {
	c.shiftReg = c.params.updateCRC(c.shiftReg, input)
	return c
}

//Result64 returns the calculated CRC result as uint64
func (c *CRC) Result64() uint64 {
	return c.params.finalizeCRC(c.shiftReg)
}
