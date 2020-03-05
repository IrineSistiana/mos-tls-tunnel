package core

import (
	"sync"

	"github.com/xtaci/smux"
)

type muxSession struct {
	mu sync.Mutex
	*smux.Session
}

type muxStream struct {
	*smux.Stream
	onClose func() error
}

func newMuxSession(s *smux.Session) *muxSession {
	return &muxSession{Session: s}
}

func (s *muxSession) openStream(maxStreamLimit int) (*muxStream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.NumStreams() >= maxStreamLimit {
		return nil, ErrTooManyStreams
	}

	stream, err := s.OpenStream()
	if err != nil {
		return nil, err
	}
	return &muxStream{Stream: stream, onClose: s.tryCloseOnIdle}, nil
}

func (s *muxSession) tryCloseOnIdle() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.NumStreams() == 0 {
		return s.Close()
	}
	return nil
}

func (s *muxStream) Close() error {
	s.Stream.Close()
	// tryCloseOnIdle
	return s.onClose()
}
