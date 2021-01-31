package waitstate

import (
	"context"
	"errors"
	"sync"
)

var ErrorClosed = errors.New("WaitState is closed")

type WaitState struct {
	sync.Mutex
	value interface{}

	updateCount uint64
	updateChan  chan (struct{})

	closed bool
}

func (w *WaitState) closeChan() {
	if w.updateChan != nil {
		close(w.updateChan)
		w.updateChan = nil
	}
}

func (w *WaitState) Set(new interface{}) {
	w.Lock()
	defer w.Unlock()

	w.value = new
	w.updateCount++
	w.closeChan()
}

func (w *WaitState) Close() {
	w.Lock()
	defer w.Unlock()
	w.closed = true
	w.closeChan()
}

func (w *WaitState) Get(ctx context.Context, checkFunc func(updateCount uint64, value interface{}) bool) (uint64, interface{}, error) {
	for {
		w.Lock()

		if w.closed {
			w.Unlock()
			return 0, nil, ErrorClosed
		}

		tmpCount := w.updateCount
		tmpValue := w.value

		if checkFunc == nil || checkFunc(w.updateCount, w.value) {
			w.Unlock()
			return tmpCount, tmpValue, nil
		}

		if w.updateChan == nil {
			w.updateChan = make(chan (struct{}))
		}
		c := w.updateChan
		w.Unlock()

		select {
		case <-ctx.Done():
			return tmpCount, tmpValue, ctx.Err()
		case <-c:
		}
	}
}

func (w *WaitState) GetNewer(ctx context.Context, lastCount uint64) (uint64, interface{}, error) {
	return w.Get(ctx, func(updateCount uint64, value interface{}) bool {
		return updateCount > lastCount
	})
}
