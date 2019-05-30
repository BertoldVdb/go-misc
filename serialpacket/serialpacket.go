package serialpacket

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/sigurn/crc8"
)

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
	messageAck    MessageType = 0xFF
	messageNack   MessageType = 0xFE
	messagePing   MessageType = 0x01
	messageIDHash MessageType = 0x02
	messageID     MessageType = 0x03
)

type commandReplyStruct struct {
	Payload []byte
	Error   error
}

type commandStruct struct {
	Packet      []byte
	Timeout     <-chan (time.Time)
	TimeoutMs   int
	Unsolicited bool
	ReplyChan   chan (commandReplyStruct)
}

type SerialPacket struct {
	sync.Mutex
	port io.ReadWriteCloser

	receiverState        receiverStateType
	receiverPacketLength uint8
	receiverPacketIndex  uint8
	receiverBuffer       [256]byte

	UnsolicitedHandler func(msgType MessageType, buf []byte)

	rxChan    chan ([]byte)
	rxChanAck chan (struct{})

	cmdChan chan (*commandStruct)

	currentCommand   *commandStruct
	compressedSerial uint8
	fullSerial       []byte
	synced           bool

	unlockKey []byte
}

func (s *SerialPacket) calcCRC(receive bool, cmd MessageType, payload []byte) (uint8, error) {
	crc := crc8.Checksum(payload, crcTable)

	if !receive {
		if cmd != messagePing && cmd != messageID {
			s.Lock()
			crc ^= s.compressedSerial
			synced := s.synced
			s.Unlock()

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

func (s *SerialPacket) sendReset() error {
	var buffer [256]byte

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

func (s *SerialPacket) sendPacket(payload []byte) error {
	pl := len(payload)

	if pl >= 256 || pl < 1 {
		return errors.New("Payload too long or too short")
	}

	/* Calculate CRC */
	var buffer = make([]byte, 0, pl+3)
	buffer = append(buffer, 'B')
	buffer = append(buffer, byte(pl))
	buffer = append(buffer, payload...)
	crc, err := s.calcCRC(false, MessageType(payload[0]), payload)
	if err != nil {
		return err
	}

	buffer = append(buffer, crc)

	_, err = s.port.Write(buffer)
	return err
}

func (s *SerialPacket) processInput(buffer []byte) error {
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
			crc, err := s.calcCRC(true, 0, receivedPacket)
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

func (s *SerialPacket) readWorker() error {
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

func (s *SerialPacket) drain(ms int) {
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

func (s *SerialPacket) processCommandReply(payload []byte, err error) {
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

func (s *SerialPacket) ProtocolHandler() error {
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

			err := s.sendPacket(cmd.Packet)
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

func (s *SerialPacket) SendCommand(cmd MessageType, payload []byte, timeout int) ([]byte, error) {
	buf := make([]byte, 1, 1+len(payload))
	buf[0] = byte(cmd)
	buf = append(buf, payload...)

	unsolicited := timeout <= 0

	cmdS := &commandStruct{
		Packet:      buf,
		Unsolicited: unsolicited,
		ReplyChan:   make(chan (commandReplyStruct), 1),
		TimeoutMs:   timeout,
	}

	s.cmdChan <- cmdS
	reply := <-cmdS.ReplyChan

	return reply.Payload, reply.Error
}

func (s *SerialPacket) GetDeviceSerial() ([]byte, error) {
	s.Lock()
	sync := s.synced
	s.Unlock()

	if !sync {
		return s.SendCommand(messageID, nil, 100)
	}

	return s.SendCommand(messageIDHash, nil, 100)
}

func (s *SerialPacket) syncTry() error {
	random := make([]byte, 16)

	for i := 0; i < 3; i++ {
		len := rand.Intn(9) + 8
		syncRandom := random[:len]
		rand.Read(syncRandom)

		reply, err := s.SendCommand(messagePing, syncRandom, 100)
		if err != nil {
			return err
		} else if bytes.Compare(reply, syncRandom) != 0 {
			return ErrorSyncFailed
		}
	}

	return nil
}

func (s *SerialPacket) Connect() ([]byte, error) {
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

func (s *SerialPacket) TestComm() error {
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

func CreateProtocol(port io.ReadWriteCloser, key []byte) *SerialPacket {
	a := &SerialPacket{}

	a.cmdChan = make(chan (*commandStruct), 20)
	a.port = port
	a.unlockKey = key

	return a
}

func (s *SerialPacket) CloseProtocol() error {
	return s.port.Close()
}
