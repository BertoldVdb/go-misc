package closeflag

import (
	"errors"
	"sync"
)

// CloseFlag is a simple object that has a close function that closes a channel and can be called many times
type CloseFlag struct {
	mutex     sync.Mutex
	closed    bool
	closeChan chan (struct{})

	// CloseFunc will be called the first time Close is called. It is allowed to call Close itself
	CloseFunc func() error
}

var (
	// ErrorClosed is returned when the object is closed multiple times
	ErrorClosed = errors.New("CloseFlag was already closed. This is harmless.")
)

// Chan returns a channel that will be closed upon closing the CloseFlag
func (c *CloseFlag) Chan() <-chan (struct{}) {
	c.mutex.Lock()
	if c.closeChan == nil {
		c.closeChan = make(chan (struct{}))
		if c.closed {
			close(c.closeChan)
		}
	}
	c.mutex.Unlock()

	return c.closeChan
}

// Close closes the CloseFlag. It can safely be called multiple times
func (c *CloseFlag) Close() error {
	c.mutex.Lock()
	closed := c.closed
	c.closed = true

	if !closed && c.closeChan != nil {
		close(c.closeChan)
	}
	c.mutex.Unlock()

	if closed {
		return ErrorClosed
	}

	if c.CloseFunc != nil {
		return c.CloseFunc()
	}

	return nil
}
