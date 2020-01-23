package framer

import (
	"testing"

	"github.com/BertoldVdb/go-misc/serialpacket/framer/testutil"
)

func testType(t *testing.T, ft string) {
	framer, err := NewFramer(ft, nil, nil)
	if err != nil {
		t.Errorf("Undesired error returned: %s", err)
		return
	}
	testutil.FramerRunTests(t, framer)
}

func TestAll(t *testing.T) {
	testType(t, "hdlc")
}

func TestBadType(t *testing.T) {
	tmp, err := NewFramer("asdfasdf", nil, nil)
	if tmp != nil {
		t.Error("Got framer for unsupported type")
	}
	if err != ErrorUnknown {
		t.Error("Invalid error returned")
	}
}
