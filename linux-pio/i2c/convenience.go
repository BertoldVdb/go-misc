package i2c

type Device struct {
	bus     *Bus
	address uint16
}

func (b *Bus) GetDevice(address uint16) *Device {
	return &Device{
		bus:     b,
		address: address,
	}
}

func (d *Device) Transfer(writeBuf []byte, readBuf []byte) error {
	return d.bus.Transfer(d.address, writeBuf, readBuf)
}

func (d *Device) WriteReg8(reg uint8, value uint8) error {
	write := []byte{reg, value}
	return d.Transfer(write, nil)
}

func (d *Device) ReadReg8(reg uint8) (uint8, error) {
	write := []byte{reg}
	read := make([]byte, 1)
	err := d.Transfer(write, read)
	if err != nil {
		return 0, err
	}
	return read[0], nil
}
