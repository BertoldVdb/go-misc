package hdlc

import (
	"testing"

	"github.com/BertoldVdb/go-misc/multicrc"
	"github.com/BertoldVdb/go-misc/serialpacket/framer/framerinterface"

	"github.com/BertoldVdb/go-misc/serialpacket/framer/testutil"
)

func testWithOptions(t *testing.T, options *framerinterface.FramerOptions, expectError bool) {
	/* Use testutil to run the test */
	framer, err := NewHDLCFramer(nil, options)
	if err != nil {
		if !expectError {
			t.Error(err)
		}
	} else {
		testutil.FramerRunTests(t, framer)
	}
}

func TestHDLC(t *testing.T) {
	testWithOptions(t, nil, false)
	testWithOptions(t, framerinterface.DefaultFramerOptions().Set(framerinterface.OptionCRCParam, multicrc.Crc32MPEG2), false)
	testWithOptions(t, framerinterface.DefaultFramerOptions().Set(framerinterface.OptionByteFrameStart, 0xAC), false)

	var empty [256]bool
	testWithOptions(t, framerinterface.DefaultFramerOptions().
		Set(framerinterface.OptionRxIgnore, empty).
		Set(framerinterface.OptionTxEscape, empty), false)

	testWithOptions(t, framerinterface.DefaultFramerOptions().Set(framerinterface.OptionByteFrameStart, 0x20), true)
}
