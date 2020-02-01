package tokenqueue

import (
	"context"
	"errors"
	"sync"
)

var (
	//ErrorClosed is returned when the queue has been closed
	ErrorClosed = errors.New("Queue has been closed")
)

//Token is an interface that a token should implement. The cleanup method is called upon closing the queue
type Token interface {
	Cleanup()
}

//Queue is a type of queue that allows requesting token (for example command slots) and passing them
//to a processor that can return them after the command has been completed
type Queue struct {
	sync.Mutex
	closed bool

	maxCapacity     int
	targetCapacity  int
	currentCapacity int

	availableTokens chan (Token)
	committedTokens chan (Token)
	discardTokens   chan (Token)
}

//TokenFactory is a function that is called when creating a queue. It should return pointers to tokens
type TokenFactory func() Token

//NewQueue creates the tokenqueue with a given maximum capacity, initial capacity and factory
func NewQueue(maximumCapacity int, initialCapacity int, factory TokenFactory) *Queue {
	q := &Queue{
		maxCapacity:     maximumCapacity,
		availableTokens: make(chan (Token), maximumCapacity),
		committedTokens: make(chan (Token), maximumCapacity),
		discardTokens:   make(chan (Token), maximumCapacity),
	}

	for i := 0; i < maximumCapacity; i++ {
		token := factory()
		if token == nil {
			return nil
		}
		q.discardTokens <- token
	}

	q.EnableDisableTokens(initialCapacity)

	return q
}

func (q *Queue) tokenFromChannel(ctx context.Context, channel chan (Token)) (Token, error) {
	select {
	case t, ok := <-channel:
		if !ok {
			return nil, ErrorClosed
		}
		return t, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

//GetAvailableToken will return a token for a free slot to the client
func (q *Queue) GetAvailableToken(ctx context.Context) (Token, error) {
	for {
		token, err := q.tokenFromChannel(ctx, q.availableTokens)

		if err != nil {
			return token, err
		}

		/* Check if tokens need to be removed from circulation */
		q.Lock()
		if q.currentCapacity > q.targetCapacity {
			q.discardTokens <- token
			q.currentCapacity--
			q.Unlock()
		} else {
			q.Unlock()
			return token, err
		}
	}
}

//GetCommittedToken will return a committed token to the processor
func (q *Queue) GetCommittedToken(ctx context.Context) (Token, error) {
	return q.tokenFromChannel(ctx, q.committedTokens)
}

func (q *Queue) getChannelReader(ctx context.Context, f func(context.Context) (Token, error)) <-chan (Token) {
	c := make(chan (Token))

	go func() {
		defer close(c)

		for {
			t, err := f(ctx)
			if err != nil {
				return
			}

			c <- t
		}
	}()

	return c
}

//GetAvailableTokenChan returns a channel from which available tokens can be read
func (q *Queue) GetAvailableTokenChan(ctx context.Context) <-chan (Token) {
	return q.getChannelReader(ctx, q.GetAvailableToken)
}

//GetCommittedTokenChan returns a channel from which committed tokens can be read
func (q *Queue) GetCommittedTokenChan(ctx context.Context) <-chan (Token) {
	return q.getChannelReader(ctx, q.GetCommittedToken)
}

func (q *Queue) tokenToChannel(channel chan (Token), t Token) error {
	q.Lock()
	defer q.Unlock()

	/* This never blocks, if the queue is closed then we put the tokens in a dedicated discard channel */
	if !q.closed {
		channel <- t
		return nil
	}

	q.discardTokens <- t
	return ErrorClosed
}

//CommitToken takes a token returned by GetAvailableToken and commits it after preparing
func (q *Queue) CommitToken(t Token) error {
	return q.tokenToChannel(q.committedTokens, t)
}

//ReleaseToken takes a token returned by GetCommittedToken and releases it after processing
func (q *Queue) ReleaseToken(t Token) error {
	return q.tokenToChannel(q.availableTokens, t)
}

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}

func drainAndCloseChannel(channel chan (Token), amount int) int {
	count := 0

	if amount < 0 {
	loop:
		for {
			select {
			case t := <-channel:
				t.Cleanup()
				count++
			default:
				break loop
			}
		}
	} else {
		for ; amount > 0; amount-- {
			t := <-channel
			t.Cleanup()
			count++
		}
	}

	close(channel)

	return count
}

//Close closes the tokenqueue
func (q *Queue) Close() {
	q.Lock()
	closed := q.closed
	q.closed = true
	q.Unlock()

	if closed {
		return
	}

	capacityRemaining := q.maxCapacity

	/* Drain the tokens in the round robbin channels */
	capacityRemaining -= drainAndCloseChannel(q.availableTokens, -1)
	capacityRemaining -= drainAndCloseChannel(q.committedTokens, -1)

	/* If a producer or consumer commits a token when the queue is closed, they are sent
	   to this channel when committed or released so they can be reclaimed */
	capacityRemaining -= drainAndCloseChannel(q.discardTokens, capacityRemaining)

	assert(capacityRemaining == 0, "capacityRemaining was not 0")
}

//EnableDisableTokens is used to temporary change the amount of tokens that are available
//Returns true if the update is possible, false if not.
//Eg. the BLE HCI command queue is dynamic
func (q *Queue) EnableDisableTokens(amount int) bool {
	q.Lock()
	defer q.Unlock()

	if q.closed {
		return false
	}

	if amount > q.maxCapacity || amount < 0 {
		return false
	}

	q.targetCapacity = amount

	/* If needed, make more available, so move n tokens from discard to available */
	for q.currentCapacity < q.targetCapacity {
		token, ok := <-q.discardTokens
		assert(ok, "DiscardTokens channel was closed, although we have the lock on q.closed and tested for it...")
		q.availableTokens <- token
		q.currentCapacity++
	}

	/* If currentCapacity > targetCapacity GetAvailableToken() will discard tokens untill this is not the case anymore */

	return true
}
