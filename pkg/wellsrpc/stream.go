package wellsrpc

import (
	"context"
	"errors"
	"sync"
)

type Stream struct {
	ID     uint32
	send   func([]byte) error
	recvCh chan []byte
	closed bool
	mu     sync.Mutex
}

func newStream(id uint32, send func([]byte) error) *Stream {
	return &Stream{
		ID:     id,
		send:   send,
		recvCh: make(chan []byte, 128), // increased buffer
	}
}

func (s *Stream) Send(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("stream closed")
	}
	return s.send(b)
}

func (s *Stream) Recv(ctx context.Context) ([]byte, error) {
	select {
	case b, ok := <-s.recvCh:
		if !ok {
			return nil, errors.New("stream closed")
		}
		return b, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Stream) Close() {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.recvCh)
	}
	s.mu.Unlock()
}
