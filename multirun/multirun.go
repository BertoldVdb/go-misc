package multirun

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/BertoldVdb/go-misc/closeflag"
)

var (
	ErrorClosed = errors.New("The multirun was closed")
)

// Runnable specifies an object with a blocking Run method and a Close method that makes Run return.
type Runnable interface {
	Run() error
	Close() error
}

// RunnableReady specifies an object with a blocking Run method and a Close method that makes Run return.
type RunnableReady interface {
	Run(func()) error
	Close() error
}

// MultiRun runs multiple Runnable objects
type MultiRun struct {
	sync.Mutex

	items          []RunnableReady
	running        []bool
	runningUpdated chan (int)
	closeflag      closeflag.CloseFlag
}

type wrapperRunnable struct {
	RunnableReady
	item Runnable
}

func (w *wrapperRunnable) Run(ready func()) error {
	ready()
	return w.item.Run()
}

func (w *wrapperRunnable) Close() error {
	return w.item.Close()
}

type wrapperFunc struct {
	sync.Mutex
	RunnableReady
	runCb   func() error
	closeCb func() error
	doClose bool
}

func (w *wrapperFunc) Run(ready func()) error {
	err := w.runCb()
	if err == nil {
		if w.closeCb != nil {
			w.Lock()
			w.doClose = true
			w.Unlock()
		}

		ready()
	}
	return err
}

func (w *wrapperFunc) Close() error {
	w.Lock()
	doClose := w.doClose
	w.doClose = false
	w.Unlock()

	if doClose {
		return w.closeCb()
	}
	return nil
}

func (m *MultiRun) RegisterRunnableReady(item RunnableReady) {
	m.items = append(m.items, item)
}

func (m *MultiRun) RegisterRunnable(item Runnable) {
	m.RegisterRunnableReady(&wrapperRunnable{item: item})
}

func (m *MultiRun) RegisterFunc(runCb func() error, closeCb func() error) {
	m.RegisterRunnableReady(&wrapperFunc{runCb: runCb, closeCb: closeCb})
}

// Run runs multiple Runnable Items and waits for all of them to complete. If one of the Runnables return
// an error, the others are also stopped.
func (m *MultiRun) Run(ready func()) error {
	/* Don't do work if we are closed already */
	if m.closeflag.IsClosed() {
		return ErrorClosed
	}

	readyChan := make(chan (struct{}), 1)
	readyFunc := func() {
		select {
		case readyChan <- struct{}{}:
		default:
		}
	}

	var wg sync.WaitGroup
	var resultMutex sync.Mutex
	var result error

	m.Lock()
	if m.running == nil {
		m.running = make([]bool, len(m.items))
		m.runningUpdated = make(chan (int), len(m.items))
	}
	m.Unlock()

loop:
	for i := range m.items {
		wg.Add(1)
		m.Lock()
		m.running[i] = true
		m.Unlock()

		go func(index int) {
			defer wg.Done()

			err := m.items[index].Run(readyFunc)
			m.runningUpdated <- index

			if err != nil {
				resultMutex.Lock()
				if result == nil {
					result = err
				}
				resultMutex.Unlock()

				m.Close()
			}
		}(i)

		select {
		/* Handle closing */
		case <-m.closeflag.Chan():
			break loop

		/* Handle ready */
		case <-readyChan:
		}
	}

	if !m.closeflag.IsClosed() {
		if ready != nil {
			ready()
		}
	}

	wg.Wait()

	if m.closeflag.IsClosed() {
		if result == nil {
			result = ErrorClosed
		}
	}

	return result

}

// Close calls the Close method on all Items.
func (m *MultiRun) Close() error {
	err := m.closeflag.Close()
	if err != nil {
		return err
	}

	/* Close in reverse order the items were started */
	for i := len(m.items) - 1; i >= 0; i-- {
		item := m.items[i]
		err2 := item.Close()
		if err == nil {
			err = err2
		}

		m.Lock()
		if m.running != nil {
			for m.running[i] {
				m.Unlock()
				index := <-m.runningUpdated
				m.Lock()
				m.running[index] = false
			}
		}
		m.Unlock()
	}

	return err
}

func (m *MultiRun) HandleSIGTERM() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		go func() {
			select {
			case <-c:
				fmt.Fprintln(os.Stderr, "Pressed ^C a second time, quitting right away.")
			case <-time.After(5 * time.Second):
				fmt.Fprintln(os.Stderr, "Timeout during shutdown, quitting with dirty state.")
			}
			os.Exit(1)
		}()
		m.Close()
	}()
}
