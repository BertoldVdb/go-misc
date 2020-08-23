package slotset

import (
	"context"
)

type stateType int

const (
	stateNotGiven = 0
	stateInactive = 1
	stateActive   = 2
	stateWaitPost = 3
)

type Slot struct {
	parent *SlotSet

	Data interface{}

	closed  bool
	errChan chan (error)
	id      int
	state   stateType
}

func (s *Slot) WaitGetChan() <-chan (error) {
	return s.errChan
}

func (s *Slot) WaitCtx(ctx context.Context) (bool, error) {
	select {
	case err, ok := <-s.errChan:
		if !ok {
			return ok, ErrorClosed
		}
		return ok, err

	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (s *Slot) PostWithoutLock(err error) {
	if s.parent.closed.IsClosed() {
		return
	}

	assert(s.state == stateInactive || s.state == stateActive || s.state == stateWaitPost, "Slot could not be posted")

	if s.state == stateWaitPost {
		s.state = stateNotGiven
		s.parent.putInternal(s)
		return
	}

	s.state = stateInactive

	select {
	case s.errChan <- err:
	default:
	}
}

func (s *Slot) Post(err error) {
	s.parent.Lock()
	defer s.parent.Unlock()

	s.PostWithoutLock(err)
}

func (s *Slot) GetID() int {
	return s.id
}

func (s *Slot) Activate() {
	s.parent.Lock()
	defer s.parent.Unlock()

	assert(s.state == stateInactive, "Slot could not be activated!")
	s.state = stateActive
}

func (s *Slot) Deactivate() {
	s.parent.Lock()
	defer s.parent.Unlock()

	assert(s.state == stateInactive || s.state == stateActive, "Slot could not be deactivated!")
	s.state = stateInactive
}

func (s *Slot) prepare() {
	s.parent.Lock()
	defer s.parent.Unlock()

	assert(s.state == stateNotGiven, "Slot was already given out!")
	s.state = stateInactive

	/* Drain the channel of spurious responses */
	for {
		select {
		case _, ok := <-s.errChan:
			if !ok {
				return
			}
		default:
			return
		}
	}
}

func (s *Slot) release() bool {
	s.parent.Lock()
	defer s.parent.Unlock()

	assert(s.state != stateNotGiven, "Slot was not yet given out!")
	if s.state == stateActive {
		s.state = stateWaitPost
		return false
	}

	s.state = stateNotGiven
	return true
}

func (ss *SlotSet) newSlot(id int) *Slot {
	return &Slot{
		parent:  ss,
		id:      id,
		errChan: make(chan (error), 1),
		state:   stateNotGiven,
	}
}
