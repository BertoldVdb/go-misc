package serial

import "io"

// Port is an extended io.ReadWriteCloser that also allows changing
// some serial port specific settings
type Port interface {
	io.ReadWriteCloser

	/* Configuration */
	SetInterfaceRate(rate uint32) error
	SetFlowControl(enabled bool) error

	/* Pins */
	SetDTR(enabled bool) error
	SetRTS(enabled bool) error
	GetPins() (PortPins, error)
}

// PortOptions is a parameter struct for Open
type PortOptions struct {
	PortName      string
	InterfaceRate uint32
	FlowControl   bool
}

// PortPins indicates the state of the modem control signals
type PortPins struct {
	DSR bool
	DTR bool
	RTS bool
	CTS bool
	DCD bool
	RNG bool
}

// Open creates an object that implements the SerialPort interface
func Open(options *PortOptions) (Port, error) {
	return openPortOs(options)
}
