package framerinterface

import (
	"io"
	"sync/atomic"
	"time"
)

// BaseStats contains statistics about the framer operating performance.
// The values containing the word Received are consistent during execution of the FramerReceivedPacketHandler.
// The values containing the word Sent are consistent when no SendPacket is executing
type BaseStats struct {
	FramesReceivedOversized     uint64
	FramesReceivedZeroLength    uint64
	FramesReceivedWrongChecksum uint64
	FramesReceivedValid         uint64

	FramesSent uint64

	BytesSent        uint64
	BytesSentEscaped uint64

	BytesReceived        uint64
	BytesReceivedEscaped uint64
}

// PacketMetadata contains information about the packet passed to the receive handler
type PacketMetadata struct {
	//RxTime is a timestamp when the first byte was received
	RxTime time.Time
}

// FramerReceivedPacketHandler is the type of callback function invoked when a packet is received
type FramerReceivedPacketHandler func(payload []byte, metadata *PacketMetadata) error

// Framer is a generic interface to send packets over a stream
type Framer interface {
	SendPacket(payload []byte) (int64, error)
	SetPort(port io.ReadWriter) error
	GetStats() BaseStats
	Run(receivedPacket FramerReceivedPacketHandler) error
}

// CopyBaseStatsAtomic makes a copy of BaseStats using atomic access
func (s *BaseStats) CopyBaseStatsAtomic() BaseStats {
	r := BaseStats{
		FramesReceivedOversized:     atomic.LoadUint64(&s.FramesReceivedOversized),
		FramesReceivedZeroLength:    atomic.LoadUint64(&s.FramesReceivedZeroLength),
		FramesReceivedWrongChecksum: atomic.LoadUint64(&s.FramesReceivedWrongChecksum),
		FramesReceivedValid:         atomic.LoadUint64(&s.FramesReceivedValid),
		FramesSent:                  atomic.LoadUint64(&s.FramesSent),
		BytesSent:                   atomic.LoadUint64(&s.BytesSent),
		BytesSentEscaped:            atomic.LoadUint64(&s.BytesSentEscaped),
		BytesReceived:               atomic.LoadUint64(&s.BytesReceived),
		BytesReceivedEscaped:        atomic.LoadUint64(&s.BytesReceivedEscaped),
	}

	return r
}

// FramerOption defines the different options that can be provided. Not every framer supports all
// options. All options are optional and sane defaults are used.
type FramerOption int

const (
	// OptionRxIgnore contains an array with 256 booleans. If the value corresponding to a received character is true, the byte is dropped
	OptionRxIgnore FramerOption = 0x0

	// OptionTxEscape contains an array with 256 booleans. If the value corresponding to a character is true, it will be escaped on TX
	OptionTxEscape FramerOption = 0x1

	// OptionTxRxAreEqual can be set to false if TX and RX use different Escape/Ignore maps. This will disable some sanity checks
	OptionTxRxAreEqual FramerOption = 0x4

	// OptionCRCParam contains a *multicrc.Params indicating the CRC type that is added. Default: no crc
	OptionCRCParam FramerOption = 0x2

	// OptionMaxPacketLen contains an integer which specifies the maximum packet length. If <=0 the length is unlimited.
	OptionMaxPacketLen FramerOption = 0x3

	// OptionByteFrameStart contains a byte indicating the start of frame delimited
	OptionByteFrameStart FramerOption = 0x100

	// OptionByteFrameEnd contains a byte indicating the frame termination symbol
	OptionByteFrameEnd FramerOption = 0x101

	// OptionByteEscape contains a byte indicating the escape symbol
	OptionByteEscape FramerOption = 0x102

	// OptionByteEscapeXOR contains a byte indicating the escape XOR value
	OptionByteEscapeXOR FramerOption = 0x103
)

// FramerOptions contains options passed to the framer constructor
type FramerOptions struct {
	configMap map[FramerOption]interface{}
}

// Get returns the value of a given option
func (o *FramerOptions) Get(t FramerOption) (interface{}, bool) {
	if o == nil || o.configMap == nil {
		return nil, false
	}

	result, ok := o.configMap[t]
	return result, ok
}

// GetDefault returns the value of a given option, returning a specified default value if it is not found
func (o *FramerOptions) GetDefault(t FramerOption, defVal interface{}) interface{} {
	value, ok := o.Get(t)

	if !ok {
		return defVal
	}

	return value
}

// GetInt returns the integer value of a given option, returning a specified default value if it is not found.
func (o *FramerOptions) GetInt(t FramerOption, defVal int) int {
	value, ok := o.Get(t)

	if !ok {
		return defVal
	}

	return value.(int)
}

// GetBool returns the boolean value of a given option, returning a specified default value if it is not found.
func (o *FramerOptions) GetBool(t FramerOption, defVal bool) bool {
	value, ok := o.Get(t)

	if !ok {
		return defVal
	}

	return value.(bool)
}

// DefaultFramerOptions returns the default framer options (note that this is currently a nil value)
func DefaultFramerOptions() *FramerOptions {
	return nil
}

// Set modifies the specified option with the given value
func (o *FramerOptions) Set(t FramerOption, value interface{}) *FramerOptions {
	if o == nil {
		o = &FramerOptions{}
	}

	if o.configMap == nil {
		o.configMap = make(map[FramerOption]interface{})
	}

	o.configMap[t] = value

	return o
}
