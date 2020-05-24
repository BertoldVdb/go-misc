package multirun

import (
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var globalMutex sync.Mutex
var isNotReady bool
var errorNewRun = errors.New("A new run method was started before we were ready")

type SimpleRunnable struct {
	t *testing.T
	RunnableReady
	sync.Mutex

	closed bool
	c      chan (struct{})

	running    bool
	readyDelay bool
}

func SimpleRunnableNew(t *testing.T, readyDelay bool) *SimpleRunnable {
	return &SimpleRunnable{
		c:          make(chan (struct{})),
		t:          t,
		readyDelay: readyDelay,
	}
}

func checkReady() bool {
	/* Check ready handling */
	valid := true
	globalMutex.Lock()
	if isNotReady {
		valid = false
	}
	isNotReady = true
	globalMutex.Unlock()
	if valid {
		time.Sleep(time.Duration(20000*rand.Float32()) * time.Microsecond)
		globalMutex.Lock()
		if !isNotReady {
			valid = false
		}
		isNotReady = false
		globalMutex.Unlock()
	}
	return valid
}

func (s *SimpleRunnable) Run(ready func()) error {
	defer func() {
		s.Lock()
		s.running = false
		s.Unlock()
	}()
	s.Lock()
	s.running = true
	s.Unlock()

	if !s.readyDelay {
		ready()
	}

	if !checkReady() {
		return errorNewRun
	}

	if s.readyDelay {
		ready()
	}

	<-s.c

	return nil
}
func (s *SimpleRunnable) Close() error {
	s.Lock()
	defer s.Unlock()

	if s.closed {
		s.t.Error("SimpleRunnable closed twice")
	} else {
		s.closed = true
		close(s.c)
	}

	return nil
}

type ErrorRunnable struct {
	runError   error
	closeError error
}

func (s *ErrorRunnable) Run() error {
	return s.runError
}
func (s *ErrorRunnable) Close() error {
	return s.closeError
}

func testBasicInternal(t *testing.T, runError int, closeError bool, readyDelay bool) {
	errorRunnable := &ErrorRunnable{}
	runErrorE := errors.New("Test Error, Run")
	if runError == 1 {
		errorRunnable.runError = runErrorE
	}
	if closeError {
		errorRunnable.closeError = errors.New("Test Error, Close")
	}

	m := &MultiRun{}
	funcCalled := false

	items := make([]*SimpleRunnable, 10)
	for i := range items {
		items[i] = SimpleRunnableNew(t, readyDelay)
		m.RegisterRunnableReady(items[i])

		if i == 3 {
			m.RegisterRunnable(errorRunnable)
		}
		if i == 6 {
			m.RegisterFunc(func() error {
				funcCalled = true
				if runError == 2 {
					return runErrorE
				}
				if !checkReady() {
					return errorNewRun
				}
				return nil
			}, nil)
		}
	}

	err := m.Run(func() {
		if !funcCalled {
			t.Error("The registered function was not called...")
		}

		if !readyDelay {
			/* It is not guaranteed when the running flag will be updated, so wait 100 ms*/
			time.Sleep(100 * time.Millisecond)
		}

		for i := range items {
			if !items[i].running {
				t.Error("Item", i, "is not running")
			}
		}
		err := m.Close()
		if err != errorRunnable.closeError {
			t.Error("Returned error from Close() was not correct", err, errorRunnable.closeError)
		}
	})

	expectedError := ErrorClosed
	if runError > 0 {
		expectedError = runErrorE
	}
	if !readyDelay {
		expectedError = errorNewRun
	}
	if err != expectedError {
		t.Error("Returned error from Run() was not correct", err, expectedError)
	}

	/* It is not guaranteed when the running flag will be updated, so wait 50 ms*/
	time.Sleep(50 * time.Millisecond)
	for i := range items {
		if items[i].running {
			t.Error("Item", i, "is still running")
		}
	}

	/* Try to close again. */
	m.Close()
}

func TestBasicFunctionality(t *testing.T) {
	/* Try with ready handling */
	testBasicInternal(t, 0, false, true)
	testBasicInternal(t, 1, false, true)
	testBasicInternal(t, 2, false, true)
	testBasicInternal(t, 0, true, true)

	/* Use instant ready */
	testBasicInternal(t, 0, false, false)
}

type SimpleRunnableNoReady struct {
	sync.Mutex
	t      *testing.T
	closed bool
	c      chan (struct{})
}

func SimpleRunnableNoReadyNew(t *testing.T) *SimpleRunnableNoReady {
	return &SimpleRunnableNoReady{
		c: make(chan (struct{})),
		t: t,
	}
}

func (s *SimpleRunnableNoReady) Run() error {
	<-s.c

	/* Take some time to close */
	time.Sleep(50 * time.Millisecond)

	return nil
}
func (s *SimpleRunnableNoReady) Close() error {
	s.Lock()

	if s.closed {
		s.t.Error("SimpleRunnableNoReady closed twice")
	} else {
		s.closed = true
		close(s.c)
	}

	s.Unlock()

	return nil
}

func TestClose(t *testing.T) {
	for i := 0; i < 10; i++ {
		m := &MultiRun{}

		if i == 0 {
			m.Close()
		} else {
			go func() {
				time.Sleep(time.Duration(100000*rand.Float32()) * time.Microsecond)
				m.Close()
			}()
		}

		for j := 0; j < 10; j++ {
			m.RegisterRunnable(SimpleRunnableNoReadyNew(t))
			if j == 5 && i < 10 {
				m.RegisterFunc(func() error {
					/* Waste some time during starting */
					time.Sleep(50 * time.Millisecond)
					return nil
				}, nil)
			}
		}

		c := make(chan (struct{}), 1)
		go func() {
			err := m.Run(nil)
			if err != ErrorClosed && err != nil {
				t.Error("Run returned error", err)
			}
			close(c)
		}()

		select {
		case <-c:
		case <-time.After(time.Second):
			t.Error("Close function did not stop Run")
			return
		}
	}
}
