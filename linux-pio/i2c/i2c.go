package i2c

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

type Bus struct {
	mutex sync.Mutex
	file  *os.File
}

func OpenBus(busID int) (*Bus, error) {
	b := &Bus{}

	var err error
	b.file, err = os.OpenFile(fmt.Sprintf("/dev/i2c-%d", busID), syscall.O_RDWR|syscall.O_NOCTTY, 0600)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (b *Bus) Transfer(address uint16, writeBuf []byte, readBuf []byte) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	const i2cFlagsRead uint16 = 1
	const i2cRdWr uintptr = 0x00000707

	type msg struct {
		Address uint16
		Flags   uint16
		Len     uint16
		Buf     uintptr
	}

	writeMsg := msg{
		Address: address,
		Flags:   0,
	}

	readMsg := msg{
		Address: address,
		Flags:   i2cFlagsRead,
	}

	var transfer []msg
	if writeBuf != nil {
		writeMsg.Len = uint16(len(writeBuf))
		writeMsg.Buf = uintptr(unsafe.Pointer(&writeBuf[0]))

		transfer = append(transfer, writeMsg)
	}
	if readBuf != nil {
		readMsg.Len = uint16(len(readBuf))
		readMsg.Buf = uintptr(unsafe.Pointer(&readBuf[0]))

		transfer = append(transfer, readMsg)
	}

	if len(transfer) == 0 {
		// A succesful, albeit useless, transfer
		return nil
	}

	type rdWrRaw struct {
		Messages    uintptr
		NumMessages uint32
	}

	param := rdWrRaw{
		Messages:    uintptr(unsafe.Pointer(&transfer[0])),
		NumMessages: uint32(len(transfer)),
	}

	_, _, errNo := syscall.Syscall(syscall.SYS_IOCTL, uintptr(b.file.Fd()), i2cRdWr, uintptr(unsafe.Pointer(&param)))

	runtime.KeepAlive(transfer)
	runtime.KeepAlive(writeBuf)
	runtime.KeepAlive(readBuf)

	if errNo != 0 {
		return fmt.Errorf("I2C transfer failed: %s", errNo.Error())
	}

	return nil
}
