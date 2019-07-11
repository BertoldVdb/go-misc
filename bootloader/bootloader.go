package bootloader

import (
	"encoding/binary"
	"io"
	"log"
	"os"

	"github.com/BertoldVdb/go-misc/serialpacket"
)

type Error string

func (e Error) Error() string { return string(e) }

const ErrrorProtocol = Error("Protocol error")
const ErrorHeaderSizeInvalid = Error("Invalid header size")
const ErrorHeaderVersionUnknown = Error("Unkown header version")
const ErrorReadFailed = Error("Read failed")
const ErrorImageTooLarge = Error("Image too large")
const ErrorImageHashInvalid = Error("Image hash invalid")
const ErrorImageHashMissing = Error("Image hash missing")
const ErrorImageSignatureLengthInvalid = Error("Image signature length invalid")
const ErrorImageSignatureInvalid = Error("Image signature invalid")
const ErrorImageSignatureMissing = Error("Image signature missing")
const ErrorDowngradeNotAllowed = Error("Downgrade rejected")
const ErrorWrongParition = Error("Wrong partition")
const ErrorParitionDoesNotExist = Error("Partition does not exist")
const ErrorImageEncryptionMissing = Error("Encryption Missing")
const ErrorBadAlignment = Error("Bad Alignment")
const ErrorUnsupportedResult = Error("Unsupported result")
const ErrorBootFailed = Error("Boot failed")

var imageErrors = []error{
	nil,
	ErrorHeaderSizeInvalid,
	ErrorHeaderVersionUnknown,
	ErrorReadFailed,
	ErrorImageTooLarge,
	ErrorImageHashInvalid,
	ErrorImageHashMissing,
	ErrorImageSignatureLengthInvalid,
	ErrorImageSignatureInvalid,
	ErrorImageSignatureMissing,
	ErrorDowngradeNotAllowed,
	ErrorWrongParition,
	ErrorParitionDoesNotExist,
	ErrorImageEncryptionMissing,
	ErrorBadAlignment,
}

func getError(result byte) error {
	if int(result) >= len(imageErrors) {
		return ErrorUnsupportedResult
	}
	return imageErrors[int(result)]
}

type Bootloader struct {
	device *serialpacket.Device
}

func (b *Bootloader) LoadImage(filename string, partition int) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	// Start upload
	pl := []byte{byte(partition)}
	_, err = b.device.SendCommand('U', pl, 500)
	if err != nil {
		return err
	}

	seqnum := 0
	fragment := make([]byte, 60)

	for {
		n, err := file.Read(fragment[1:])
		if err != nil && err != io.EOF {
			return err
		}
		fragment[0] = byte(seqnum)

		for try := 0; ; try++ {
			log.Printf("Uploading %d bytes to target device (attempt: %d, sequence: %d)", n, try+1, seqnum)
			reply, err := b.device.SendCommand('b', fragment[:(n+1)], 500)

			if err == nil {
				if len(reply) == 2 {
					if reply[1] != 255 {
						return getError(reply[1])
					}
					if reply[0] == byte(seqnum+1) {
						seqnum++
						break
					}
				}
			}

			if try == 3 {
				if err != nil {
					return err
				}

				return ErrrorProtocol
			}
		}
	}
}

func (b *Bootloader) GetSecureCounter() (uint32, error) {
	reply, err := b.device.SendCommand('d', nil, 500)
	if err != nil {
		return 0, err
	}

	if len(reply) != 4 {
		return 0, ErrrorProtocol
	}

	return binary.BigEndian.Uint32(reply), nil
}

func (b *Bootloader) CheckImage(partition int) (uint32, error) {
	pl := []byte{byte(partition)}
	reply, err := b.device.SendCommand('v', pl, 500)
	if err != nil {
		return 0, err
	}

	if len(reply) != 1 && len(reply) != 5 {
		return 0, ErrrorProtocol
	}

	version := uint32(0)
	if len(reply) == 5 {
		version = binary.BigEndian.Uint32(reply[1:])
	}

	return version, getError(reply[0])
}

func (b *Bootloader) SetAddress(addr uint8) error {
	pl := []byte{byte(addr)}
	_, err := b.device.SendCommand('A', pl, 500)

	return err
}

func (b *Bootloader) Boot(partition int) error {
	pl := []byte{byte(partition)}
	_, err := b.device.SendCommand('G', pl, 500)
	if err == nil {
		/* The device will not respond, since it will jump to the app code */
		return ErrorBootFailed
	}
	return nil
}

func (b *Bootloader) SetSecure() error {
	_, err := b.device.SendCommand('s', nil, 500)
	return err
}

func NewBootloader(device *serialpacket.Device) *Bootloader {
	b := &Bootloader{}

	b.device = device

	return b
}
