package framer

import (
	"errors"
	"io"
	"strings"

	"github.com/BertoldVdb/go-misc/serialpacket/framer/framerinterface"
	"github.com/BertoldVdb/go-misc/serialpacket/framer/hdlc"
)

var (
	// ErrorUnknown is returned by NewFramer when an unsupported type is requested
	ErrorUnknown = errors.New("Framer type is not supported")
)

// NewFramer creates a framer with the specified type and options. You need to pass the io.ReadWriter that will be used to transfer data.
// Current supported types are: HDLC
func NewFramer(framerType string, port io.ReadWriter, options *framerinterface.FramerOptions) (framerinterface.Framer, error) {
	switch strings.ToUpper(framerType) {
	case "HDLC":
		return hdlc.NewHDLCFramer(port, options)
	default:
		return nil, ErrorUnknown
	}
}
