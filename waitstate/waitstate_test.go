package waitstate

import (
	"context"
	"testing"
	"time"
)

func TestGetWithoutSet(t *testing.T) {
	w := WaitState{}

	/* Instant return for nil checkFunc */
	_, _, err := w.Get(context.Background(), nil)
	if err != nil {
		t.Error("Returned error", err)
	}

	/* Expired context returns with error */
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := 0
	_, _, err = w.Get(ctx, func(updateCount uint64, value interface{}) bool {
		called++
		return false
	})
	if err != ctx.Err() {
		t.Error("Expired context did not cause error")
	}
	if called != 1 {
		t.Error("Checkfunc was not called the correct amount of times ")
	}

	/* If checkfunc is happy, return instant */
	_, _, err = w.Get(context.Background(), func(updateCount uint64, value interface{}) bool {
		called++
		return true
	})
	if err != nil {
		t.Error("Returned error", err)
	}
	if called != 2 {
		t.Error("Checkfunc was not called the correct amount of times ")
	}

	/* Get after close returns ErrorClosed */
	w.Close()
	_, _, err = w.Get(context.Background(), nil)
	if err != ErrorClosed {
		t.Error("Returned wrong error after close", err)
	}
}

func TestGetWithSet(t *testing.T) {
	w := WaitState{}

	w.Set(uint64(1))
	w.Set(uint64(2))

	go func() {
		for i := 3; i < 7; i++ {
			time.Sleep(100 * time.Millisecond)
			w.Set(uint64(i))
		}
	}()

	updateCount, value, err := w.Get(context.Background(), func(updateCount uint64, value interface{}) bool {
		vu := value.(uint64)
		if vu != updateCount {
			t.Error("Value or updatecount was wrong")
		}
		return vu == 5
	})
	if err != nil {
		t.Error("Returned error", err)
	}

	if updateCount != 5 {
		t.Error("UpdateCount is wrong")
	}
	if value.(uint64) != 5 {
		t.Error("Value is wrong")
	}

	/* Get a later one */
	updateCount, value, err = w.GetNewer(context.Background(), updateCount)
	if err != nil {
		t.Error("Returned error", err)
	}
	if updateCount != 6 {
		t.Error("UpdateCount is wrong")
	}
	if value.(uint64) != 6 {
		t.Error("Value is wrong")
	}

	/* The next one will timeout */
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	updateCount, value, err = w.GetNewer(ctx, updateCount)
	cancel()

	if err != ctx.Err() {
		t.Error("Did not return context error", err)
	}
	if updateCount != 6 {
		t.Error("UpdateCount is wrong")
	}
	if value.(uint64) != 6 {
		t.Error("Value is wrong")
	}

	/* The final one will be closed while waiting */
	go func() {
		time.Sleep(200 * time.Millisecond)
		w.Close()
	}()

	updateCount, value, err = w.GetNewer(context.Background(), updateCount)
	if err != ErrorClosed {
		t.Error("Did not return ErrorClosed")
	}
	if updateCount != 0 {
		t.Error("UpdateCount was not zero")
	}
	if value != nil {
		t.Error("Value not nil")
	}
}
