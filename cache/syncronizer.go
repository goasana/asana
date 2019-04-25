package cache

import (
	"sync"
	"time"
)

type Synchronizer struct {
	memories []Cache
}

func NewSynchronizer(caches ...Cache) *Synchronizer {
	return &Synchronizer{
		memories: caches,
	}
}

func (s *Synchronizer) Get(key string, timeout time.Duration) interface{} {
	results := make(chan interface{}, len(s.memories))
	someIsNull := false

	var val interface{}
	for _, c := range s.memories {
		go func(c Cache) {
			results <- c.Get(key)
		}(c)

		select {
		case a := <-results:
			if a != nil {
				val = a
			} else {
				someIsNull = true
			}
		case <-time.After(8 * time.Millisecond):
			continue
		}
	}

	go func() {
		if someIsNull && val != nil {
			s.Put(key, val, timeout)
		}
	}()

	return val
}

func (s *Synchronizer) Put(key string, val interface{}, timeout time.Duration) error {
	var wg sync.WaitGroup

	defer wg.Wait()

	wg.Add(len(s.memories))

	err := make(chan error, len(s.memories))
	for _, c := range s.memories {
		go func(wg *sync.WaitGroup, c Cache) {
			defer wg.Done()
			err <- c.Put(key, val, timeout)
		}(&wg, c)
	}

	return <- err
}

func (s *Synchronizer) Delete(key string) {
	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(len(s.memories))

	for _, c := range s.memories {
		go func(wg *sync.WaitGroup, c Cache) {
			defer wg.Done()
			_ = c.Delete(key)
		}(&wg, c)
	}
}

func (s *Synchronizer) IsExist(key string) bool {
	m := make(chan bool, len(s.memories))

	for _, c := range s.memories {
		go func(m chan bool, c Cache) {
			exists := c.IsExist(key)
			m <- exists
		}(m, c)

		select {
		case a := <-m:
			if a {
				return a
			}
		}
	}

	return false
}

func (s *Synchronizer) ClearAll() {
	for _, c := range s.memories {
		go func(c Cache) {
			_ = c.ClearAll()
		}(c)
	}
}
