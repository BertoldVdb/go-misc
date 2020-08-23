package slotset

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func check(t *testing.T, condition bool, reason ...interface{}) {
	if !condition {
		t.Error(reason...)
		t.FailNow()
	}
}

func getExpiredContext() context.Context {
	ctxExpired, cancel := context.WithCancel(context.Background())
	cancel()
	return ctxExpired
}

func TestGet(t *testing.T) {
	ss := New(7, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	for i := 0; i < 7; i++ {
		slot, err := ss.Get(ctx)
		check(t, err == nil && slot != nil, "TestGet: Invalid error response")
	}

	slot, err := ss.Get(ctx)
	check(t, err != nil && slot == nil, "TestGet: Invalid response", err, slot)

	ss.Close()
}

func testMulti(t *testing.T, max int, workers int) {
	ss := New(max, nil)

	running := int32(0)

	var wg sync.WaitGroup
	work := func() {
		defer wg.Done()

		for i := 0; i < 20; i++ {
			slot, err := ss.Get(context.Background())
			new := atomic.AddInt32(&running, 1)
			check(t, err == nil, err)

			time.Sleep(time.Duration(rand.Float32()*1000) * time.Microsecond)

			check(t, int(new) <= max, "Too many slots returned")

			atomic.AddInt32(&running, -1)
			ss.Put(slot)
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go work()
	}
	wg.Wait()

	ss.Close()
}

func TestMulti(t *testing.T) {
	testMulti(t, 1, 1)
	testMulti(t, 1, 5)
	testMulti(t, 1, 20)
	testMulti(t, 3, 1)
	testMulti(t, 3, 5)
	testMulti(t, 3, 20)
	testMulti(t, 5, 1)
	testMulti(t, 5, 5)
	testMulti(t, 5, 20)
}

func TestMasterSlave(t *testing.T) {
	ss := New(1, nil)
	txChan := make(chan (int), 20)

	slave := func() {
		for {
			value, ok := <-txChan
			if !ok {
				return
			}

			if value == -1 {
				/* Don't respond */
				continue
			}
			if value == -2 {
				ss.Close()
				return
			}

			posted := false
			err := ss.IterateActive(func(slot *Slot) (bool, error) {
				if slot.GetID() == value {
					check(t, !posted, "Already posted")
					slot.PostWithoutLock(nil)
					posted = true
				}
				return true, nil
			})
			if err != nil {
				return
			}
			check(t, posted, "Couldn't post, activation did not work")
		}
	}

	submit := func(timeout bool, close bool, expectError bool) {
		slot, err := ss.Get(context.Background())
		if !expectError {
			check(t, err == nil, err)
		}
		if slot == nil {
			return
		}

		// In a real case we would configure the slot before activating

		// Activate the slot
		slot.Activate()

		// Do a dummy TX
		if timeout {
			txChan <- -1
		} else if close {
			txChan <- -2
		} else {
			txChan <- slot.GetID()
		}

		// Wait for slot to be processed
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*20)
		ok, err := slot.WaitCtx(ctx)
		cancel()

		if !expectError {
			check(t, ok && err == nil, "Clean case, wrong return from Wait", ok, err, timeout, close, expectError)
		} else {
			check(t, !ok && err != nil, "Error case, wrong return from Wait", ok, err, timeout, close, expectError)
		}

		// Deactivate slot
		slot.Deactivate()

		// In a real case we would read out the results from the slot

		// Return the slot as we are done
		ss.Put(slot)
	}

	go slave()

	submit(false, false, false)
	submit(false, false, false)
	submit(true, false, true)
	submit(false, false, false)
	submit(false, true, true)
	submit(false, false, true)
	submit(false, false, true)

	ss.Close()

}

func checkPanic(t *testing.T) {
	r := recover()
	if r == nil {
		t.Errorf("The code did not panic")
	}
}

func TestWeirdAbuse(t *testing.T) {
	func() {
		defer checkPanic(t)
		ss := New(7, nil)
		s1, _ := ss.Get(context.Background())
		ss.Put(&Slot{parent: ss})
		ss.Put(s1)
	}()

	func() {
		defer checkPanic(t)
		ss := New(7, nil)
		ss2 := New(7, nil)

		s2, _ := ss2.Get(context.Background())
		ss.Put(s2)
	}()

	func() {
		defer checkPanic(t)
		ss := New(7, nil)
		s1, _ := ss.Get(context.Background())
		ss.Put(s1)
		s1.Post(nil)
	}()
}

func TestClosed(t *testing.T) {
	ss := New(7, nil)
	s1, _ := ss.Get(context.Background())
	ss.Close()
	s1.Post(nil)
	ss.Put(s1)
}

func TestGetChan(t *testing.T) {
	ss := New(7, nil)
	s1, _ := ss.Get(context.Background())
	ch := s1.WaitGetChan()
	check(t, ch != nil, "Channel was nil")
}

func TestInitCb(t *testing.T) {
	cnt := 0
	ss := New(7, func(slot *Slot) {
		cnt++
	})
	check(t, cnt == 7, "Function not called 7 time")
	ss.Close()
}

func TestIterate(t *testing.T) {
	ss := New(1, nil)
	ss.Close()
	err := ss.IterateActive(func(slot *Slot) (bool, error) {
		return true, nil
	})
	check(t, err == ErrorClosed, "Could iterate on closed set")

	ss = New(2, nil)
	s1, _ := ss.Get(context.Background())
	s2, _ := ss.Get(context.Background())
	s1.Activate()
	s2.Activate()

	cnt := 0
	err = ss.IterateActive(func(slot *Slot) (bool, error) {
		cnt++
		return false, nil
	})
	check(t, cnt == 1, "Count is not 1", cnt)
	check(t, err == nil, "Error returned", err)

	cnt = 0
	err = ss.IterateActive(func(slot *Slot) (bool, error) {
		cnt++
		return true, nil
	})
	check(t, cnt == 2, "Count is not 2", cnt)
	check(t, err == nil, "Error returned", err)

	cnt = 0
	testError := errors.New("Error")
	err = ss.IterateActive(func(slot *Slot) (bool, error) {
		cnt++
		return true, testError
	})
	check(t, cnt == 1, "Count is not 1", cnt)
	check(t, err == testError, "Wrong error returned", err)
}

func TestAssert(t *testing.T) {
	assert(true, "Works great")
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	assert(false, "Assert failed")
}
