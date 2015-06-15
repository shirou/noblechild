package noblechild

import (
	"errors"
	"sync"
)

type subscriber struct {
	sub map[uint16]subscribefn
	mu  *sync.Mutex
}

type subscribefn func([]byte, error)

func newSubscriber() *subscriber {
	return &subscriber{
		sub: make(map[uint16]subscribefn),
		mu:  &sync.Mutex{},
	}
}

func (s *subscriber) subscribe(h uint16, f subscribefn) {
	s.mu.Lock()
	s.sub[h] = f
	s.mu.Unlock()
}

func (s *subscriber) unsubscribe(h uint16) {
	s.mu.Lock()
	delete(s.sub, h)
	s.mu.Unlock()
}

func (s *subscriber) fn(h uint16) subscribefn {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sub[h]
}

var (
	ErrInvalidLength = errors.New("invalid length")
)
