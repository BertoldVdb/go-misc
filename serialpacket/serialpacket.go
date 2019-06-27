package serialpacket

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/sigurn/crc8"
)

const AddressDefault = 0xFF

var crcTable *crc8.Table

func init() {
	crcParam := crc8.Params{
		Poly: 0x9B,
		Init: 0x12,
		Name: "CRC-8/Packet",
	}
	crcTable = crc8.MakeTable(crcParam)
}

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrorTimeout      = Error("Command Timeout")
	ErrorNack         = Error("Command rejected")
	ErrorSyncFailed   = Error("Invaild sync response")
	ErrorNotConnected = Error("Not connected")
)

type receiverStateType int

const (
	waitSync   receiverStateType = 0
	readLength receiverStateType = 1
	readPacket receiverStateType = 2
	readCRC    receiverStateType = 3
)

type MessageType byte

const (
	messageAck     MessageType = 0xFF
	messageNack    MessageType = 0xFE
	messagePing    MessageType = 0x01
	messageIDHash  MessageType = 0x02
	messageID      MessageType = 0x03
	messageSysTime MessageType = 0x04
)

type commandReplyStruct struct {
	Payload []byte
	Error   error
}

type commandStruct struct {
	Device *Device

	Packet      []byte
	Timeout     <-chan (time.Time)
	TimeoutMs   int
	Unsolicited bool
	ReplyChan   chan (commandReplyStruct)
}

type Bus struct {
	port io.ReadWriteCloser

	receiverState        receiverStateType
	receiverPacketLength uint8
	receiverPacketIndex  uint8
	receiverBuffer       [256]byte

	UnsolicitedHandler func(msgType MessageType, buf []byte)

	rxChan    chan ([]byte)
	rxChanAck chan (struct{})

	cmdChan chan (*commandStruct)

	currentCommand *commandStruct

	unlockKey []byte
}

type Device struct {
	sync.Mutex

	bus *Bus

	compressedSerial uint8
	fullSerial       []byte
	synced           bool

	address uint8
}

func (s *Bus) GetDefaultDevice() *Device {
	return &Device{
		address: AddressDefault,
		bus:     s,
	}
}

func (s *Bus) GetDevice(address uint8) *Device {
	return &Device{
		address: address,
		bus:     s,
	}
}

func (s *Bus) calcCRC(d *Device, cmd MessageType, payload []byte) (uint8, error) {
	crc := crc8.Checksum(payload, crcTable)

	if d != nil {
		if cmd != messagePing && cmd != messageID {
			d.Lock()
			crc ^= d.compressedSerial
			synced := d.synced
			d.Unlock()

			if !synced {
				return 0, ErrorNotConnected
			}
		}
	}

	if crc == 0 {
		crc = 0xAA
	}

	return crc, nil
}

func (s *Bus) sendReset() error {
	var buffer [258]byte

	_, err := s.port.Write(buffer[:])

	if err != nil {
		return err
	}
	s.drain(100)

	if s.unlockKey != nil {
		_, err := s.port.Write(s.unlockKey)
		if err != nil {
			return err
		}
		s.drain(100)

		_, err = s.port.Write(buffer[:])
		if err != nil {
			return err
		}
		s.drain(100)
	}

	return nil
}

func (s *Device) sendPacket(payload []byte) error {
	pl := len(payload)

	if pl >= 256 || pl < 1 {
		return errors.New("Payload too long or too short")
	}

	/* Calculate CRC */
	var buffer = make([]byte, 0, pl+3)
	buffer = append(buffer, 'B')
	if s.address != 0xFF {
		buffer = append(buffer, s.address)
	}
	buffer = append(buffer, byte(pl))
	buffer = append(buffer, payload...)
	crc, err := s.bus.calcCRC(s, MessageType(payload[0]), payload)
	if err != nil {
		return err
	}

	buffer = append(buffer, crc)

	_, err = s.bus.port.Write(buffer)
	return err
}

func (s *Bus) processInput(buffer []byte) error {
	for _, m := range buffer {
		switch s.receiverState {
		case waitSync:
			if m == 'B' {
				s.receiverState = readLength
			}

		case readLength:
			s.receiverPacketLength = m
			s.receiverPacketIndex = 0

			if m == 0 {
				s.receiverState = waitSync
			} else {
				s.receiverState = readPacket
			}

		case readPacket:
			s.receiverBuffer[s.receiverPacketIndex] = m
			s.receiverPacketIndex++
			if s.receiverPacketIndex == s.receiverPacketLength {
				s.receiverState = readCRC
			}

		case readCRC:
			receivedPacket := s.receiverBuffer[:s.receiverPacketLength]
			crc, err := s.calcCRC(nil, 0, receivedPacket)
			if err != nil {
				return err
			}
			if crc == m {
				msgType := MessageType(receivedPacket[0])

				if msgType == messageAck {
					s.processCommandReply(receivedPacket[1:], nil)
				} else if msgType == messageNack {
					s.processCommandReply(receivedPacket[1:], ErrorNack)
				} else if s.UnsolicitedHandler != nil {
					s.UnsolicitedHandler(msgType, receivedPacket[1:])
				}
			}
			s.receiverState = waitSync
		}
	}

	return nil
}

func (s *Bus) readWorker() error {
	defer close(s.rxChan)

	var buffer [64]byte

	for {
		n, err := s.port.Read(buffer[:])
		if err != nil {
			return err
		}

		s.rxChan <- buffer[:n]
		_, ok := <-s.rxChanAck
		if !ok {
			break
		}
	}

	return nil
}

func (s *Bus) drain(ms int) {
	timeout := time.After(time.Duration(ms) * time.Millisecond)

	for {
		select {
		case <-timeout:
			return
		case _, ok := <-s.rxChan:
			if !ok {
				return
			}

			s.rxChanAck <- struct{}{}
		}
	}
}

func (s *Bus) processCommandReply(payload []byte, err error) {
	if err != nil {
		s.sendReset()
	}

	if s.currentCommand != nil {
		if s.currentCommand.ReplyChan != nil {
			s.currentCommand.ReplyChan <- commandReplyStruct{
				Error:   err,
				Payload: payload,
			}
		}
		s.currentCommand = nil
	}
}

func (s *Bus) ProtocolHandler() error {
	s.rxChan = make(chan ([]byte))
	s.rxChanAck = make(chan (struct{}))
	defer close(s.rxChanAck)

	/* Reset line */
	s.sendReset()

	/* Start receiver */
	s.receiverState = waitSync
	go s.readWorker()

loop:
	for {
		cmdInChan := s.cmdChan
		var timeoutChan <-chan (time.Time)
		if s.currentCommand != nil {
			cmdInChan = nil
			timeoutChan = s.currentCommand.Timeout
		}

		select {
		case buffer, ok := <-s.rxChan:
			if !ok {
				break loop
			}

			err := s.processInput(buffer)
			if err != nil {
				return err
			}

			s.rxChanAck <- struct{}{}

		case <-timeoutChan:
			s.processCommandReply(nil, ErrorTimeout)

		case cmd := <-cmdInChan:
			s.currentCommand = cmd

			err := cmd.Device.sendPacket(cmd.Packet)
			if err != nil {
				s.processCommandReply(nil, err)
			} else if cmd.Unsolicited {
				s.processCommandReply(nil, nil)
			} else {
				s.currentCommand.Timeout = time.After(time.Duration(cmd.TimeoutMs) * time.Millisecond)
			}
		}
	}

	return nil
}

func (s *Device) SendCommand(cmd MessageType, payload []byte, timeout int) ([]byte, error) {
	buf := make([]byte, 1, 1+len(payload))
	buf[0] = byte(cmd)
	buf = append(buf, payload...)

	unsolicited := timeout <= 0

	cmdS := &commandStruct{
		Packet:      buf,
		Device:      s,
		Unsolicited: unsolicited,
		ReplyChan:   make(chan (commandReplyStruct), 1),
		TimeoutMs:   timeout,
	}

	s.bus.cmdChan <- cmdS
	reply := <-cmdS.ReplyChan

	return reply.Payload, reply.Error
}

func (s *Device) GetDeviceSerial() ([]byte, error) {
	s.Lock()
	sync := s.synced
	s.Unlock()

	if !sync {
		return s.SendCommand(messageID, nil, 1000)
	}

	return s.SendCommand(messageIDHash, nil, 1000)
}

func (s *Device) GetSystemTime() (uint64, error) {
	reply, err := s.SendCommand(messageSysTime, nil, 1000)
	if err != nil {
		return 0, err
	}

	if len(reply) == 4 {
		return uint64(binary.BigEndian.Uint32(reply)), nil
	}
	if len(reply) == 8 {
		return binary.BigEndian.Uint64(reply), nil
	}

	// Not supported
	return 0, nil
}

func (s *Device) syncTry() error {
	random := make([]byte, 16)

	for i := 0; i < 3; i++ {
		len := rand.Intn(9) + 8
		syncRandom := random[:len]
		rand.Read(syncRandom)

		reply, err := s.SendCommand(messagePing, syncRandom, 1000) //TODO make configurable
		if err != nil {
			return err
		} else if bytes.Compare(reply, syncRandom) != 0 {
			return ErrorSyncFailed
		}
	}

	return nil
}

func (s *Device) Connect() ([]byte, error) {
	var retVal error

	s.Lock()
	s.synced = false
	s.Unlock()

	for i := 0; i < 3; i++ {
		retVal = s.syncTry()
		if retVal == nil {
			break
		}
	}

	if retVal != nil {
		return nil, retVal
	}

	serial, err := s.GetDeviceSerial()
	if serial != nil && err == nil {
		compressedSerial := uint8(0)
		for _, m := range serial {
			compressedSerial ^= m
		}

		s.Lock()
		s.compressedSerial = compressedSerial
		s.fullSerial = serial
		s.synced = true
		s.Unlock()
	}

	if err != nil {
		return nil, err
	}

	return serial, nil
}

func (s *Device) TestComm() error {
	serial2, err := s.GetDeviceSerial()
	if err != nil {
		return err
	}

	s.Lock()
	serial := []byte(nil)
	if s.synced {
		serial = s.fullSerial
	}
	s.Unlock()

	if serial == nil {
		return ErrorNotConnected
	}

	if bytes.Compare(serial, serial2) != 0 {
		return ErrorNack
	}

	return nil
}

func CreateProtocol(port io.ReadWriteCloser, key []byte) *Bus {
	a := &Bus{}

	a.cmdChan = make(chan (*commandStruct), 20)
	a.port = port
	a.unlockKey = key

	return a
}

func (s *Bus) CloseProtocol() error {
	return s.port.Close()
}
