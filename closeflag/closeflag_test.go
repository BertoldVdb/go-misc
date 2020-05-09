package closeflag

import (
	"errors"
	"log"
	"sync"
	"testing"
	"time"
)

func testChannel(ch <-chan (struct{})) bool {
	select {
	case <-ch:
		return true
	case <-time.After(time.Millisecond * 50):
		return false
	}
}

func testInternal(t *testing.T, useChannel bool, useChannelFirst bool, useFunction bool) {
	c := CloseFlag{}

	called := false
	errorFromFunc := errors.New("Error from func")

	if useFunction {
		c.CloseFunc = func() error {
			if called {
				t.Error("Close function closed multiple times")
			}
			called = true
			return errorFromFunc
		}
	}

	if useChannelFirst {
		if testChannel(c.Chan()) {
			t.Error("Channel was already closed")
		}
	}

	if useFunction {
		if c.Close() != errorFromFunc {
			t.Error("First close did not return errorFromFunc")
		}
	} else {
		if c.Close() != nil {
			t.Error("First close did not return nil")
		}
	}

	if useFunction && !called {
		t.Error("Close function was not called")
	}

	if useChannel {
		if !testChannel(c.Chan()) {
			t.Error("Channel was not closed")
		}
	}

	for i := 0; i < 10; i++ {
		if c.Close() != ErrorClosed {
			t.Error("Nect close did not return ErrorClosed")
		}
	}
}

func TestClose(t *testing.T) {
	for i := 0; i < 8; i++ {
		testInternal(t, i&1 > 0, i&2 > 0, i&4 > 0)
	}
}

func TestMultiClose(t *testing.T) {
	c := CloseFlag{}

	called := false
	c.CloseFunc = func() error {
		/* Check for deadlock issues */
		c.Close()

		if called {
			t.Error("Close func called multiple times")
		}
		called = true
		return nil
	}

	var wg sync.WaitGroup

	/* Start 16 goroutines waiting on the flag */
	wg.Add(16)
	for i := 0; i < 16; i++ {
		go func(i int) {
			defer wg.Done()
			defer c.Close()

			time.Sleep(time.Duration(i) * 10 * time.Millisecond)
			log.Println("Started listener", i)
			defer log.Println("Closing listener", i)

			select {
			case <-time.After(time.Second):
				t.Error("Did not get close signal!")
			case <-c.Chan():
			}
		}(i)
	}

	/* Start 16 goroutines that will call close */
	wg.Add(16)
	for i := 0; i < 16; i++ {
		go func() {
			defer wg.Done()
			defer c.Close()

			time.Sleep(75 * time.Millisecond)
		}()
	}

	wg.Wait()

	if !called {
		t.Error("Close func not called")
	}
}
