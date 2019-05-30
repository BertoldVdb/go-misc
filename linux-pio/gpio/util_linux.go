package gpio

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

func ioctlPtr(f *os.File, function uintptr, data unsafe.Pointer) error {
	_, _, errNo := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()),
		function,
		uintptr(data),
	)
	if errNo != 0 {
		return fmt.Errorf("IOCTL failed: %s", errNo.Error())
	}

	return nil
}

func bytesToString(input []byte) string {
	return strings.TrimRight(string(input), "\x00")
}

func stringToBytes(input string, output []byte) {
	n := copy(output, input)

	if n >= len(output) {
		n = len(output) - 1
	}

	// Null terminate string
	output[n] = 0
}
