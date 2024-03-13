package util

import (
	"sync"
)

type ChanMux[T any] struct {
	chans       []chan T
	closed      bool
	OnEmpty     func()
	OnSubscribe func()
	sync.Mutex
}

func NewChanMux[T any](source chan T) *ChanMux[T] {
	self := &ChanMux[T]{
		chans:       []chan T{},
		closed:      false,
		OnEmpty:     nil,
		OnSubscribe: nil,
	}

	go self.main(source)

	return self
}

func (self *ChanMux[T]) main(source chan T) {
	for item := range source {
		self.Lock()
		for _, c := range self.chans {
			c <- item
		}
		self.Unlock()
	}

	self.Lock()
	defer self.Unlock()
	for _, c := range self.chans {
		close(c)
	}
	self.closed = true
}

func (self *ChanMux[T]) Subscribe(buffer int) chan T {
	if self.closed {
		return nil
	}
	self.Lock()
	defer self.Unlock()

	c := make(chan T, buffer)
	self.chans = append(self.chans, c)

	if len(self.chans) == 1 && self.OnSubscribe != nil {
		self.OnSubscribe()
	}

	return c
}

func (self *ChanMux[T]) Unsubscribe(subscriber chan T) {
	if self.closed {
		return
	}
	self.Lock()
	defer self.Unlock()

	for i, c := range self.chans {
		if c == subscriber {
			self.chans = append(self.chans[:i], self.chans[i+1:]...)
			close(c)
			if len(self.chans) == 0 && self.OnEmpty != nil {
				self.OnEmpty()
			}
			return
		}
	}
}

func (self *ChanMux[T]) IsClosed() bool {
	return self.closed
}
