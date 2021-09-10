package spi

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

type Device struct {
	mutex     sync.Mutex
	file      *os.File
	Frequency uint32
}

func OpenDevice(busID int, deviceID int) (*Device, error) {
	d := &Device{
		Frequency: 1000000,
	}

	var err error
	d.file, err = os.OpenFile(fmt.Sprintf("/dev/spidev%d.%d", busID, deviceID), syscall.O_RDWR|syscall.O_NOCTTY, 0600)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func getIoctlId(numTransfers int) uintptr {
	const base uint32 = 0x40006B00

	return uintptr(base + uint32(numTransfers*0x200000))
}

func (d *Device) Transfer(writeBuf []byte, readBuf []byte) error {

	d.mutex.Lock()
	defer d.mutex.Unlock()

	type iocTransferRaw struct {
		TxBuf       uint64
		RxBuf       uint64
		Len         uint32
		Frequency   uint32
		DelayUs     uint16
		BitsPerWord uint8
		CsChange    uint8
		Pad         uint32
	}

	tr := iocTransferRaw{
		Frequency:   d.Frequency,
		DelayUs:     20,
		BitsPerWord: 8,
	}

	if writeBuf != nil {
		tr.TxBuf = uint64(uintptr(unsafe.Pointer(&writeBuf[0])))
		tr.Len = uint32(len(writeBuf))
	}
	if readBuf != nil {
		tr.RxBuf = uint64(uintptr(unsafe.Pointer(&readBuf[0])))
		tr.Len = uint32(len(readBuf))
	}

	if tr.TxBuf == 0 && tr.RxBuf == 0 {
		return nil
	}
	if tr.TxBuf != 0 && tr.RxBuf != 0 {
		if len(readBuf) != len(writeBuf) {
			return errors.New("Buffer length does not match")
		}
	}

	_, _, errNo := syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.file.Fd()), getIoctlId(1), uintptr(unsafe.Pointer(&tr)))

	runtime.KeepAlive(writeBuf)
	runtime.KeepAlive(readBuf)

	if errNo != 0 {
		return fmt.Errorf("SPI transfer failed: %s", errNo.Error())
	}

	return nil
}
