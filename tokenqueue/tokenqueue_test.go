package tokenqueue

import (
	"context"
	"log"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

type Command struct {
	active  uint32
	cleaned uint32

	totalCleaned *uint32
	t            *testing.T
}

func (c *Command) Cleanup() {
	if atomic.SwapUint32(&c.cleaned, 1) != 0 {
		c.t.Error("Token was already cleaned up")
	}

	if c.totalCleaned != nil {
		atomic.AddUint32(c.totalCleaned, 1)
	}

	log.Println("Cleaning up token")
}

func runTestCheckCleanup(capacity int, t *testing.T, work func(t *testing.T, q *Queue)) {
	totalCleaned := uint32(0)

	q := NewQueue(capacity, func() Token {
		return &Command{t: t, totalCleaned: &totalCleaned}
	})

	work(t, q)

	if int(totalCleaned) != capacity {
		t.Error("Not all tokens cleaned")
	}
}

func TestTokenQueueWorkerClient(t *testing.T) {
	runTestCheckCleanup(10, t, func(t *testing.T, q *Queue) {
		waitChan := make(chan (int), 50)

		for i := 0; i < 10; i++ {
			go testWorker(t, waitChan, i, q)
		}
		for i := 10; i < 20; i++ {
			go testClient(t, waitChan, i, q)
		}

		time.Sleep(5 * time.Second)

		/* Test multiple closes */
		q.Close()
		q.Close()
		q.Close()
		q.Close()

		maxWait := time.After(5 * time.Second)
		for i := 20; i > 0; i-- {
			q.Close()

			select {
			case <-waitChan:
			case <-maxWait:
				t.Error("Not all worker closed correctly")
				return
			}
		}
	})
}

func testWorker(test *testing.T, waitChain chan (int), id int, q *Queue) {
	defer func() { waitChain <- id }()
	ctx := context.Background()

	for {
		log.Println(id, "Requesting committed token")
		t, err := q.GetCommittedToken(ctx)
		if err != nil {
			log.Println(id, "Error in worker", err)
			break
		}
		log.Println(id, "Got committed token, working with it")

		if atomic.SwapUint32(&t.(*Command).active, 0) != 1 {
			test.Error("Token was not active")
		}

		time.Sleep(time.Duration(rand.Uint32()%200) * time.Millisecond)
		log.Println(id, "Done working on committed token, releasing")
		q.ReleaseToken(t)
		log.Println(id, "Release done")
	}
}

func testClient(test *testing.T, waitChain chan (int), id int, q *Queue) {
	defer func() { waitChain <- id }()
	ctx := context.Background()

	for {
		log.Println(id, "Requesting available token")
		t, err := q.GetAvailableToken(ctx)
		if err != nil {
			log.Println(id, "Error in client", err)
			break
		}
		log.Println(id, "Got available token, working on it")

		if atomic.SwapUint32(&t.(*Command).active, 1) != 0 {
			test.Error("Token was already active")
		}

		time.Sleep(time.Duration(rand.Uint32()%200) * time.Millisecond)
		log.Println(id, "Done working on available token, committing")
		q.CommitToken(t)
		log.Println(id, "Commit done")
	}
}

func TestTokenQueueUnUsed(t *testing.T) {
	runTestCheckCleanup(10, t, func(t *testing.T, q *Queue) {
		q.Close()
	})
}

func TestTokenQueueBadFactory(t *testing.T) {
	q := NewQueue(5, func() Token {
		return nil
	})

	if q != nil {
		t.Error("Constructor did not return nil while factory did")
	}
}

func TestAssert(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	assert(false, "Assert failed")
}

func TestTimeout(t *testing.T) {
	q := NewQueue(1, func() Token {
		return &Command{t: t}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := q.GetCommittedToken(ctx)
	if err == nil {
		t.Error("Context timeout failed")
	}

	q.Close()
}

func measureCapcity(q *Queue) int {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	count := 0
	for {
		token, err := q.GetAvailableToken(ctx)
		if err != nil {
			break
		}
		q.CommitToken(token)
		count++
	}

	ctx = context.Background()

	for i := 0; i < count; i++ {
		token, err := q.GetCommittedToken(ctx)
		if err != nil {
			break
		}
		q.ReleaseToken(token)
	}

	return count
}

func testChangeCapacity(t *testing.T, q *Queue, capacity int) {
	if !q.EnableDisableTokens(capacity) {
		t.Error("Capacity change failed")
	}
	newCapacity := measureCapcity(q)
	if newCapacity != capacity {
		t.Errorf("Asked to change capacity to %d but the new capacity was %d", capacity, newCapacity)
	}
}

func TestEnableDisable(t *testing.T) {
	q := NewQueue(10, func() Token {
		return &Command{t: t}
	})

	if measureCapcity(q) != 10 {
		t.Error("Initial capacity was not correct")
	}

	testChangeCapacity(t, q, 10)
	testChangeCapacity(t, q, 8)
	testChangeCapacity(t, q, 3)
	testChangeCapacity(t, q, 2)
	testChangeCapacity(t, q, 7)
	testChangeCapacity(t, q, 1)
	testChangeCapacity(t, q, 10)
	testChangeCapacity(t, q, 0)
	testChangeCapacity(t, q, 7)

	if q.EnableDisableTokens(11) {
		t.Error("Could change capacity to 11")
	}

	if q.EnableDisableTokens(-1) {
		t.Error("Could change capacity to -1")
	}

	q.Close()

	if q.EnableDisableTokens(5) {
		t.Error("Could change capacity to 5 after closing")
	}
}
