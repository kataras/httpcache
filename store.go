// The MIT License (MIT)
//
// Copyright (c) 2016 GeekyPanda
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package httpcache

import (
	"sync"
	"time"
)

type (

	// Store is the interface of the cache bug, default is memory store for performance reasons
	Store interface {
		// Set adds an entry to the cache by its key
		// entry must contain a valid status code, conten type, a body and optional, the expiration duration
		Set(key string, statusCode int, cType string, body []byte, expiration time.Duration)
		// Get returns an entry based on its key
		Get(key string) *Entry
		// Remove removes an entry from the cache
		Remove(key string)
		// Clear removes all entries from the cache
		Clear()
	}

	// Entry is the entry type of the cached items
	// contains the http status code, content type the full cached response body(any type of it)
	// and the expiration datetime
	Entry struct {
		StatusCode  int
		ContentType string
		Body        []byte
		// we could have a new Timer foreach cache Entry in order to be persise on the expiration but this will cost us a lot of performance,
		// (the ticker should be stopped if delete or key ovveride and so on...)
		// but I chosen to just have a generic timer with its tick on the lowest 'expires' of all cache entries that cache keeps
		Expires time.Time
	}

	// memoryStore keeps the cache bag, by default httpcache package provides one global default cache service  which provides these functions:
	// `httpcache.Cache`, `httpcache.Invalidate` and `httpcache.Start`
	// Store and NewStore used only when you want to have two different separate cache bags
	memoryStore struct {
		cache map[string]*Entry
		mu    sync.RWMutex
		once  sync.Once
	}
)

// NewMemoryStore returns a new memory store for the cache ,
// note that httpcache package provides one global default cache service  which provides these functions:
// `httpcache.Cache`, `httpcache.Invalidate` and `httpcache.Start`
//
// If you use only one global cache for all of your routes use the `httpcache.New` instead
func NewMemoryStore(gcDuration time.Duration) Store {
	s := &memoryStore{
		cache: make(map[string]*Entry),
		mu:    sync.RWMutex{},
	}
	s.startGC(gcDuration)
	return s
}

// Set adds an entry to the cache by its key
// entry must contain a valid status code, conten type, a body and optional, the expiration duration
func (s *memoryStore) Set(key string, statusCode int, cType string, body []byte, expiration time.Duration) {
	e := &Entry{
		StatusCode:  validateStatusCode(statusCode),
		ContentType: validateContentType(cType),
		Body:        body,
		Expires:     time.Now().Add(validateCacheDuration(expiration)),
	}

	s.mu.Lock()
	s.cache[key] = e
	s.mu.Unlock()
}

// Get returns an entry based on its key
func (s *memoryStore) Get(key string) *Entry {
	s.mu.RLock()
	if v, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		if time.Now().After(v.Expires) { // we check for expiration, the gc clears the cache but gc maybe late
			s.Remove(key)
			return nil
		}
		return v
	}
	s.mu.RUnlock()
	return nil
}

// Remove removes an entry from the cache
func (s *memoryStore) Remove(key string) {
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()
}

func (s *memoryStore) Clear() {
	s.mu.Lock()
	for k := range s.cache {
		delete(s.cache, k)
	}

	s.mu.Unlock()
}

// startGC starts the GC of the cache bag it should be called at the last, before the http server's listens
// Note that you must NOT call it if `httpcache.New` is used
//
// If you use only one global cache for all of your routes use the `httpcache.New` instead
func (s *memoryStore) startGC(gcDuration time.Duration) {
	if gcDuration > minimumAllowedCacheDuration {
		s.once.Do(func() {
			// start the timer to check for expirated cache entries
			tick := time.Tick(gcDuration)
			go func() {
				for range tick {
					now := time.Now()
					s.mu.Lock()
					for k, v := range s.cache {
						if now.After(v.Expires) {
							delete(s.cache, k)
						}
					}
					s.mu.Unlock()
				}
			}()
		})

	}
}
