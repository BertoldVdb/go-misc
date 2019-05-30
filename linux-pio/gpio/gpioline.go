package gpio

import (
	"errors"
	"unsafe"
)

func (gl *Lines) Close() {
	gl.file.Close()
}

type handleDataRaw struct {
	values [64]uint8
}

func (gl *Lines) SetValues(values []bool) error {
	sd := handleDataRaw{}

	for i, b := range values {
		if i >= int(gl.numLines) {
			return errors.New("Line index out of range")
		}

		if b {
			sd.values[i] = 1
		}
	}

	err := ioctlPtr(gl.file, gpiohandleSetLineValuesIoctl, unsafe.Pointer(&sd))
	if err != nil {
		return err
	}

	return nil
}

func (gl *Lines) GetValues() ([]bool, error) {
	gd := handleDataRaw{}

	err := ioctlPtr(gl.file, gpiohandleGetLineValuesIoctl, unsafe.Pointer(&gd))
	if err != nil {
		return nil, err
	}

	output := make([]bool, gl.numLines)
	for i := uint32(0); i < gl.numLines; i++ {
		output[i] = gd.values[i] > 0
	}

	return output, nil
}

func (gl *Lines) SetValue(value bool) error {
	return gl.SetValues([]bool{value})
}

func (gl *Lines) GetValue() (bool, error) {
	output, err := gl.GetValues()
	if output == nil || err != nil {
		return false, err
	}

	return output[0], err
}
