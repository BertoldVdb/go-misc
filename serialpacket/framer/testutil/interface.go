package testutil

import (
	"bytes"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/BertoldVdb/go-misc/serialpacket/framer/framerinterface"
)

func testLoopback(t *testing.T, loopback io.WriteCloser, framer framerinterface.Framer) {
	rxChan := make(chan (bytes.Buffer), 512)
	framerDone := make(chan (error), 1)

	go func() {
		framerDone <- framer.Run(func(packet []byte, pkt *framerinterface.PacketMetadata) error {
			var copy bytes.Buffer
			copy.Write(packet)

			rxChan <- copy

			return nil
		})
	}()

	for cnt := 0; cnt < 100; cnt++ {
		log.Println(cnt)

		/* Send garbage */
		loopback.Write(RandomBytes(512))
		/* Send real packet */
		packet := RandomBytes(128)
		framer.SendPacket(packet)
		/* Send garbage */
		loopback.Write(RandomBytes(128))

		timeout := time.After(time.Second)
	waitLoop:
		for {
			select {
			case rx := <-rxChan:
				if bytes.Equal(rx.Bytes(), packet) {
					break waitLoop
				}

			case <-timeout:
				t.Error("Did not rececive valid packet")
				return
			}
		}
	}

	loopback.Close()
	err := <-framerDone
	if err != io.EOF {
		t.Error("Wrong error returned after closing", err)
	}

	stats := framer.GetStats()

	if stats.FramesSent != 100 {
		t.Error("framesSent is wrong")
	}
	if stats.BytesSentEscaped < stats.BytesSent {
		t.Error("bytesSent relation is wrong")
	}
	if stats.BytesReceivedEscaped < stats.BytesReceived {
		t.Error("bytesReceived relation is wrong")
	}
}

func testErrorInHandler(t *testing.T, loopback io.WriteCloser, framer framerinterface.Framer) {
	testError := errors.New("All is well")
	framerDone := make(chan (error), 1)

	go func() {
		framerDone <- framer.Run(func(packet []byte, pkt *framerinterface.PacketMetadata) error {
			return testError
		})
	}()

	framer.SendPacket([]byte{1, 2, 3})

	err := <-framerDone
	if err != testError {
		t.Error("Wrong error returned from handler", err)
	}
}

// FramerRunTests is an internal function that will run the tests on the given framer
func FramerRunTests(t *testing.T, framer framerinterface.Framer) {
	loopback := NewLoopback()
	framer.SetPort(loopback)
	testLoopback(t, loopback, framer)

	loopback = NewLoopback()
	framer.SetPort(loopback)
	testErrorInHandler(t, loopback, framer)
}
