package once

import (
	"context"
	"sync"
)

type Once struct {
	Handler func()

	sync.Mutex
	waitChan chan (struct{})
	running  bool
	runDone  bool
}

// Wait waits for the handler function to complete. If the function is not running
// it will be triggered. If the function is being executed the first time after Reset(),
// all callers will block in Wait, until it is completed.
func (o *Once) Wait(ctx context.Context) error {
	o.Lock()
	if !o.runDone {
		w := o.waitChan
		if w != nil {
			o.Unlock()
			select {
			case <-w:
			case <-ctx.Done():
				return ctx.Err()
			}
			o.Lock()
		} else {
			o.waitChan = make(chan struct{})

			o.triggerInternal()
		}
	}
	o.Unlock()
	return nil
}

func (o *Once) triggerInternal() {
	if o.running {
		return
	}
	o.running = true
	o.Unlock()
	o.Handler()
	o.Lock()
	o.running = false
	o.runDone = true

	if o.waitChan != nil {
		close(o.waitChan)
		o.waitChan = nil
	}
}

// Trigger will start executing the handler function
func (o *Once) Trigger() {
	o.Lock()
	o.triggerInternal()
	o.Unlock()
}

// Reset will rearm Wait()
func (o *Once) Reset() {
	o.Lock()
	o.runDone = false
	o.Unlock()
}
