package slotset

import (
	"context"
	"errors"
	"sync"

	"github.com/BertoldVdb/go-misc/closeflag"
)

type SlotSet struct {
	sync.Mutex

	closed    closeflag.CloseFlag
	slotQueue chan (*Slot)
	slotSlice []*Slot
}

var (
	ErrorClosed = errors.New("SlotSet closed")
)

func New(numSlots int, initCb func(*Slot)) *SlotSet {
	s := &SlotSet{
		slotQueue: make(chan (*Slot), numSlots),
		slotSlice: make([]*Slot, numSlots),
	}

	for i := range s.slotSlice {
		slot := s.newSlot(i)
		if initCb != nil {
			initCb(slot)
		}
		s.slotSlice[i] = slot
		s.slotQueue <- slot
	}

	return s
}

func (s *SlotSet) Get(ctx context.Context) (*Slot, error) {
	select {
	case slot := <-s.slotQueue:
		slot.prepare()
		return slot, nil

	case <-s.closed.Chan():
		return nil, ErrorClosed

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *SlotSet) putInternal(slot *Slot) {
	if !s.closed.IsClosed() {
		select {
		case s.slotQueue <- slot:
		default:
			panic("Too many slots were returned")
		}
	}
}

func (s *SlotSet) Put(slot *Slot) {
	if slot.release() {
		s.putInternal(slot)
	}
}

type IterateCallback func(slot *Slot) (bool, error)

func (s *SlotSet) IterateActive(cb IterateCallback) error {
	if s.closed.IsClosed() {
		return ErrorClosed
	}

	s.Lock()
	defer s.Unlock()

	for _, m := range s.slotSlice {
		if m.state == stateActive || m.state == stateWaitPost {
			cont, err := cb(m)
			if !cont || err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SlotSet) Close() error {
	err := s.closed.Close()
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	for _, m := range s.slotSlice {
		close(m.errChan)
	}

	/* Drain the channel to ensure error at Get(). */
	select {
	case <-s.slotQueue:
	default:
	}

	return nil
}

func assert(condition bool, reason string) {
	if !condition {
		panic(reason)
	}
}
