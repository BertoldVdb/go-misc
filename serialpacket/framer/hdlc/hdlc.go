package hdlc

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BertoldVdb/go-misc/multicrc"
	"github.com/BertoldVdb/go-misc/serialpacket/framer/framerinterface"
)

// HDLC is a packet framer that implements the HDLC protocol
type HDLC struct {
	port         io.ReadWriter
	maxPacketLen int

	sendBuffer struct {
		sync.Mutex
		data bytes.Buffer
		crc  *multicrc.CRC
	}

	stats framerinterface.BaseStats

	TxCharsEscape [256]bool
	RxCharsIgnore [256]bool

	crcParams *multicrc.Params

	frameStart     byte
	frameEnd       byte
	frameEscape    byte
	frameEscapeXOR byte
}

// NewHDLCFramer is used to create a HDLC framer
func NewHDLCFramer(port io.ReadWriter, options *framerinterface.FramerOptions) (*HDLC, error) {
	s := &HDLC{
		port:           port,
		crcParams:      options.GetDefault(framerinterface.OptionCRCParam, multicrc.CrcNone).(*multicrc.Params),
		maxPacketLen:   options.GetInt(framerinterface.OptionMaxPacketLen, 256),
		frameStart:     byte(options.GetInt(framerinterface.OptionByteFrameStart, 0x7E)),
		frameEnd:       byte(options.GetInt(framerinterface.OptionByteFrameEnd, 0x7E)),
		frameEscape:    byte(options.GetInt(framerinterface.OptionByteEscape, 0x7D)),
		frameEscapeXOR: byte(options.GetInt(framerinterface.OptionByteEscapeXOR, 0x20)),
	}

	for i := 0; i < 0x20; i++ {
		s.RxCharsIgnore[i] = true
		s.TxCharsEscape[i] = true
	}

	if value, ok := options.Get(framerinterface.OptionRxIgnore); ok {
		v2 := value.([256]bool)
		copy(s.RxCharsIgnore[:], v2[:])
	}

	if value, ok := options.Get(framerinterface.OptionTxEscape); ok {
		v2 := value.([256]bool)
		copy(s.TxCharsEscape[:], v2[:])
	}

	/* Create CRC module for sender */
	s.sendBuffer.crc = multicrc.NewCRC(s.crcParams)

	/* These bytes must be escaped for the protocol to work */
	s.TxCharsEscape[s.frameEnd] = true
	s.TxCharsEscape[s.frameStart] = true
	s.TxCharsEscape[s.frameEscape] = true

	if options.GetBool(framerinterface.OptionTxRxAreEqual, true) {
		/* Sanity check escaping */
		for raw := 0; raw < len(s.TxCharsEscape); raw++ {
			if s.TxCharsEscape[raw] {
				escaped := uint8(raw) ^ s.frameEscapeXOR
				if s.RxCharsIgnore[escaped] {
					return nil, fmt.Errorf("Requested to escape char that will be ignored: %02X -> %02X", raw, escaped)
				}
			}
		}
	}

	return s, nil
}

func (s *HDLC) writeEscaped(payload []byte) {
	for _, m := range payload {
		if s.TxCharsEscape[m] {
			s.sendBuffer.data.WriteByte(s.frameEscape)
			s.sendBuffer.data.WriteByte(m ^ s.frameEscapeXOR)
		} else {
			s.sendBuffer.data.WriteByte(m)
		}
	}
}

// SendPacket is used to send a packet to the port using HDLC framing
func (s *HDLC) SendPacket(payload []byte) (int64, error) {
	s.sendBuffer.Lock()
	defer s.sendBuffer.Unlock()
	defer s.sendBuffer.data.Reset()

	s.sendBuffer.data.WriteByte(s.frameStart)
	s.writeEscaped(payload)
	var crcBuf [8]byte
	s.writeEscaped(s.sendBuffer.crc.Reset().AddBytes(payload).ResultBytes(crcBuf[:], false))
	s.sendBuffer.data.WriteByte(s.frameEnd)

	n, err := s.sendBuffer.data.WriteTo(s.port)

	if n > 0 {
		nu := uint64(n)
		iu := uint64(len(payload))
		if iu > nu {
			iu = nu
		}

		atomic.AddUint64(&s.stats.FramesSent, 1)
		atomic.AddUint64(&s.stats.BytesSent, iu)
		atomic.AddUint64(&s.stats.BytesSentEscaped, nu)
	}

	return n, err
}

// SetPort can be used to change the port used by the framer. It may not be executed concurrently
// with Run
func (s *HDLC) SetPort(port io.ReadWriter) error {
	s.sendBuffer.Lock()
	defer s.sendBuffer.Unlock()

	s.port = port

	return nil
}

// Run should be called to start the receiver process. It will only return
// on read errors (eg, port closed)
func (s *HDLC) Run(receivedPacket framerinterface.FramerReceivedPacketHandler) error {
	var tmpBuf [512]byte
	var rxBuffer bytes.Buffer

	isEscaped := false
	isValid := true
	isFirst := true

	reset := func() {
		isValid = true
		isEscaped = false
		isFirst = true

		rxBuffer.Reset()
	}

	var firstByteTimestamp time.Time

	crc := multicrc.NewCRC(s.crcParams)

	for {
		n, err := s.port.Read(tmpBuf[:])
		if err != nil {
			return err
		}

		for _, m := range tmpBuf[:n] {
			atomic.AddUint64(&s.stats.BytesReceivedEscaped, 1)

			if isFirst {
				firstByteTimestamp = time.Now()
				isFirst = false
			}

			if m == s.frameEnd {
				if rxBuffer.Len() > 0 {
					atomic.AddUint64(&s.stats.BytesReceived, uint64(rxBuffer.Len()))

					if isValid && !isEscaped {
						atomic.AddUint64(&s.stats.FramesReceivedValid, 1)

						message := rxBuffer.Bytes()
						if len(message) < crc.ResultLenBytes() {
							atomic.AddUint64(&s.stats.FramesReceivedWrongChecksum, 1)
						} else {
							crcIndex := len(message) - crc.ResultLenBytes()

							var crcCalcBuf [8]byte
							if bytes.Equal(crc.Reset().AddBytes(message[:crcIndex]).ResultBytes(crcCalcBuf[:], false), message[crcIndex:]) {
								pkt := framerinterface.PacketMetadata{
									RxTime: firstByteTimestamp,
								}

								err := receivedPacket(message[:crcIndex], &pkt)
								if err != nil {
									return err
								}
							} else {
								atomic.AddUint64(&s.stats.FramesReceivedWrongChecksum, 1)
							}
						}
					}
				} else {
					atomic.AddUint64(&s.stats.FramesReceivedZeroLength, 1)
				}

				reset()

			} else if m == s.frameStart {
				reset()

			} else if s.RxCharsIgnore[m] {
			} else if isEscaped {
				isEscaped = false

				if isValid {
					rxBuffer.WriteByte(m ^ s.frameEscapeXOR)
				}

			} else if m == s.frameEscape {
				isEscaped = true

			} else if isValid {
				rxBuffer.WriteByte(m)
			}

			if isValid && s.maxPacketLen > 0 && rxBuffer.Len() > s.maxPacketLen {
				atomic.AddUint64(&s.stats.FramesReceivedOversized, 1)
				isValid = false
			}
		}
	}
}

// GetStats returns a safely accessed snapshot of the statistics
func (s *HDLC) GetStats() framerinterface.BaseStats {
	return s.stats.CopyBaseStatsAtomic()
}
