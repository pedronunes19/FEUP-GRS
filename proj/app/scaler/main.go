package scaler

import "sync"

func Run(s *sync.WaitGroup) {
	s.Done()
}