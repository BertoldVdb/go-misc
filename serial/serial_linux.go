package serial

import (
	"errors"
	"io"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type serialPortLinux struct {
	file      *os.File
	closeChan chan (struct{})
}

func (port *serialPortLinux) SetFlowControl(enabled bool) error {
	termios, err := unix.IoctlGetTermios(int(port.file.Fd()), unix.TCGETS2)
	if err != nil {
		return err
	}

	if enabled {
		termios.Cflag |= unix.CRTSCTS
	} else {
		termios.Cflag &= ^uint32(unix.CRTSCTS)
	}

	return unix.IoctlSetTermios(int(port.file.Fd()), unix.TCSETS2, termios)
}

func (port *serialPortLinux) SetInterfaceRate(rate uint32) error {
	termios, err := unix.IoctlGetTermios(int(port.file.Fd()), unix.TCGETS2)
	if err != nil {
		return err
	}

	termios.Cflag &= ^uint32(unix.CBAUD)
	termios.Cflag |= uint32(unix.BOTHER)
	termios.Ispeed = rate
	termios.Ospeed = rate

	return unix.IoctlSetTermios(int(port.file.Fd()), unix.TCSETS2, termios)
}

func (port *serialPortLinux) defaultPortConfig() error {
	termios := &unix.Termios{}
	/* Most basic serial config possible */
	termios.Cflag |= uint32(syscall.CS8 | syscall.CLOCAL | syscall.CREAD)

	/* Calling close during read does not always seem to cancel it (usually does though)
	 * Therefore we put a 1s timeout so reads always return.
	 * According to the go documentation this should not be neccesary
	 * (Although I don't know if a serial port counts as a normal linux file) */
	termios.Cc[syscall.VTIME] = 10
	termios.Cc[syscall.VMIN] = 0

	/* Set it */
	return unix.IoctlSetTermios(int(port.file.Fd()), unix.TCSETS2, termios)
}

func openPortOs(options *PortOptions) (*serialPortLinux, error) {
	file, err := os.OpenFile(options.PortName, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}

	port := &serialPortLinux{}
	port.file = file
	port.closeChan = make(chan (struct{}), 1)
	port.closeChan <- struct{}{}

	/* Set default termios */
	err = port.defaultPortConfig()
	if err != nil {
		goto failed
	}

	err = port.SetInterfaceRate(options.InterfaceRate)
	if err != nil {
		goto failed
	}

	err = port.SetFlowControl(options.FlowControl)
	if err != nil {
		goto failed
	}

	err = unix.SetNonblock(int(port.file.Fd()), false)
	if err != nil {
		goto failed
	}

	return port, nil

failed:
	file.Close()
	return nil, err
}

func (port *serialPortLinux) setPinIoctl(enabled bool, pin int) error {
	req := unix.TIOCMBIC
	if enabled {
		req = unix.TIOCMBIS
	}

	r, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(port.file.Fd()), uintptr(req), uintptr(unsafe.Pointer(&pin)))

	if err != 0 || r < 0 {
		return os.NewSyscallError("TIOCMBIC/TIOCMBIS", err)
	}
	return nil
}

func (port *serialPortLinux) SetDTR(enabled bool) error {
	return port.setPinIoctl(enabled, unix.TIOCM_DTR)
}

func (port *serialPortLinux) SetRTS(enabled bool) error {
	return port.setPinIoctl(enabled, unix.TIOCM_RTS)
}

func (port *serialPortLinux) GetPins() (PortPins, error) {
	pins := PortPins{}

	var v int
	r, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(port.file.Fd()), uintptr(unix.TIOCMGET), uintptr(unsafe.Pointer(&v)))

	if err != 0 || r < 0 {
		return pins, os.NewSyscallError("TIOCMBIC/TIOCMBIS", err)
	}

	/* Decode response */
	pins.DTR = (v & unix.TIOCM_DTR) > 0
	pins.RTS = (v & unix.TIOCM_RTS) > 0
	pins.CTS = (v & unix.TIOCM_CTS) > 0
	pins.DCD = (v & unix.TIOCM_CAR) > 0
	pins.RNG = (v & unix.TIOCM_RNG) > 0
	pins.DSR = (v & unix.TIOCM_DSR) > 0

	return pins, nil
}

func (port *serialPortLinux) Read(p []byte) (int, error) {
	for {
		token, ok := <-port.closeChan
		if !ok {
			return 0, errors.New("Port closed")
		}

		n, err := port.file.Read(p)
		port.closeChan <- token

		if err != io.EOF {
			return n, err
		}
	}
}

func (port *serialPortLinux) Write(p []byte) (int, error) {
	return port.file.Write(p)
}

func (port *serialPortLinux) Close() error {
	_, ok := <-port.closeChan
	if ok {
		close(port.closeChan)
		return port.Close()
	}

	return nil
}
