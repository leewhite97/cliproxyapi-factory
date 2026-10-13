package provider

import (
	"fmt"
)

func (s *Service) registerStream(id string, host StreamHost) error {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	if s.stopped {
		return fmt.Errorf("factory: provider shutting down")
	}
	if id == "" {
		return fmt.Errorf("factory: missing upstream stream ID")
	}
	if _, ok := s.streams[id]; ok {
		return fmt.Errorf("factory: duplicate upstream stream ID")
	}
	s.wg.Add(1)
	s.streams[id] = host
	return nil
}
func (s *Service) releaseStream(id string) {
	s.streamMu.Lock()
	delete(s.streams, id)
	s.streamMu.Unlock()
	s.wg.Done()
}
