package events

import (
	"sync"

	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/event"
)

var (
	ForkDetected    = &FeedOf[struct{}]{}
	NewChainHead    = &FeedOf[*types.Block]{}
	NewMinedBlock   = &FeedOf[*types.Block]{}
	NewNetworkBlock = &FeedOf[*types.Block]{}
	NewTx           = &FeedOf[types.Tx]{}
	SyncStarted     = &FeedOf[struct{}]{}
	SyncFinished    = &FeedOf[struct{}]{}
)

type Callback[T any] func(data T)

type Subscription[T any] struct {
	s event.Subscription
	c chan T
	w *sync.WaitGroup
}

type FeedOf[T any] struct {
	once sync.Once
	feed event.FeedOf[T]

	subscriptions map[string]*Subscription[T]
}

func (e *FeedOf[T]) Send(data T) (sent int) {
	return e.feed.Send(data)
}

func (e *FeedOf[T]) init() {
	e.subscriptions = make(map[string]*Subscription[T])
}

func (e *FeedOf[T]) Subscribe(id string, callback Callback[T]) {
	e.once.Do(e.init)

	e.Unsubscribe(id)
	sub := &Subscription[T]{c: make(chan T), w: &sync.WaitGroup{}}
	sub.s = e.feed.Subscribe(sub.c)
	sub.w.Add(1)
	go func() {
		defer sub.w.Done()
		for {
			select {
			case t := <-sub.c:
				callback(t)
			case <-sub.s.Err():
				return
			}
		}
	}()
	e.subscriptions[id] = sub
}

func (e *FeedOf[T]) Unsubscribe(id string) *sync.WaitGroup {
	e.once.Do(e.init)

	sub, ok := e.subscriptions[id]
	if ok {
		delete(e.subscriptions, id)
		sub.s.Unsubscribe()
		return sub.w
	}
	return &sync.WaitGroup{}
}
