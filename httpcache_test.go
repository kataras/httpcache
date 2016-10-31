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

package httpcache_test

import (
	"github.com/gavv/httpexpect"
	"github.com/geekypanda/go-errors"
	"github.com/geekypanda/httpcache"
	"github.com/geekypanda/httpcache/httptest"
	"github.com/valyala/fasthttp"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

var (
	remotescheme       = "http://"
	httpremoteaddr     = "127.0.0.1:8888"
	fasthttpremoteaddr = "127.0.0.1:9999"
	cacheDuration      = 10 * time.Second
	serverSleepDur     = 5 * time.Second
	expectedBodyStr    = "Imagine it as a big message to achieve x20 response performance!"
	errTestFailed      = errors.New("Expected the main handler to be executed %d times instead of %d.")
)

// ~14secs
func runTest(e *httpexpect.Expect, counterPtr *uint32, expectedBodyStr string) error {
	e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	time.Sleep(2 * time.Second) // lets wait for a while because saving to cache is going on a goroutine, for performance reasons
	e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	counter := atomic.LoadUint32(counterPtr)
	if counter > 1 {
		// n should be 1 because it doesn't changed after the first call
		return errTestFailed.Format(1, counter)
	}
	time.Sleep(cacheDuration)

	// cache should be cleared now
	e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	time.Sleep(2 * time.Second)
	// let's call again , the cache should be saved
	e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	counter = atomic.LoadUint32(counterPtr)
	if counter != 2 {
		return errTestFailed.Format(2, counter)
	}

	return nil
}

func TestCachePackageLevel(t *testing.T) {
	httpcache.Clear()
	mux := http.NewServeMux()
	var n uint32
	mux.Handle("/", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddUint32(&n, 1)
		res.Write([]byte(expectedBodyStr))
	}))

	cachedMux := httpcache.Cache(mux, cacheDuration)
	e := httptest.New(t, httptest.Handler(cachedMux))
	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}

}

func TestCachePackageLevelDistributed(t *testing.T) {
	httpcache.Clear()
	// start the remote cache service
	go http.ListenAndServe(httpremoteaddr, httpcache.Default)
	time.Sleep(serverSleepDur) // let's wait a little
	mux := http.NewServeMux()

	var n uint32

	mux.Handle("/", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddUint32(&n, 1)
		res.Write([]byte(expectedBodyStr))
	}))

	cachedMux := httpcache.Request(remotescheme+httpremoteaddr, mux, cacheDuration)
	e := httptest.New(t, httptest.Handler(cachedMux))
	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}
}

func TestCachePackageLevelFasthttp(t *testing.T) {
	httpcache.Clear()
	var n uint32
	mux := func(reqCtx *fasthttp.RequestCtx) {
		atomic.AddUint32(&n, 1)
		reqCtx.Write([]byte(expectedBodyStr))
	}

	cachedMux := httpcache.Fasthttp.Cache(mux, cacheDuration)
	e := httptest.New(t, httptest.RequestHandler(cachedMux))
	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}
}

func TestCachePackageLevelDistributedFasthttp(t *testing.T) {
	httpcache.Clear()
	go fasthttp.ListenAndServe(fasthttpremoteaddr, httpcache.Fasthttp.ServeHTTP)
	time.Sleep(serverSleepDur)
	var n uint32
	mux := func(reqCtx *fasthttp.RequestCtx) {
		atomic.AddUint32(&n, 1)
		reqCtx.Write([]byte(expectedBodyStr))
	}

	cachedMux := httpcache.RequestFasthttp(remotescheme+fasthttpremoteaddr, mux, cacheDuration)
	e := httptest.New(t, httptest.RequestHandler(cachedMux))

	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}
}

/*

func TestCacheParallel(t *testing.T) {
	t.Parallel()

	store := httpcache.NewMemoryStore(httpcache.GC)
	service := httpcache.NewService(store)

	mux := http.NewServeMux()
	var n uint32
	mux.Handle("/", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddUint32(&n, 1)
		res.Write([]byte(expectedBodyStr))
	}))

	cachedMux := service.Cache(mux, cacheDuration)
	e := httptest.New(t, httptest.Handler(cachedMux))
	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}

}

func TestCacheDistributedParallel(t *testing.T) {
	t.Parallel()
	store := httpcache.NewMemoryStore(httpcache.GC)
	serverService := httpcache.NewService(store)

	// start the remote cache service
	go http.ListenAndServe(httpremoteaddr, serverService)
	time.Sleep(serverSleepDur) // let's wait a little
	mux := http.NewServeMux()

	var n uint32

	mux.Handle("/", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddUint32(&n, 1)
		res.Write([]byte(expectedBodyStr))
	}))

	cachedMux := httpcache.Request(remotescheme+httpremoteaddr, mux, cacheDuration)
	e := httptest.New(t, httptest.Handler(cachedMux))
	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}
}

func TestCacheFasthttpParallel(t *testing.T) {
	t.Parallel()
	store := httpcache.NewMemoryStore(httpcache.GC)
	service := httpcache.NewServiceFasthttp(store)

	var n uint32
	mux := func(reqCtx *fasthttp.RequestCtx) {
		atomic.AddUint32(&n, 1)
		reqCtx.Write([]byte(expectedBodyStr))
	}

	cachedMux := service.Cache(mux, cacheDuration)
	e := httptest.New(t, httptest.RequestHandler(cachedMux))
	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}
}

func TestCacheDistributedFasthttpParallel(t *testing.T) {
	t.Parallel()

	store := httpcache.NewMemoryStore(httpcache.GC)
	serverService := httpcache.NewServiceFasthttp(store)

	go fasthttp.ListenAndServe(fasthttpremoteaddr, serverService.ServeHTTP)
	time.Sleep(serverSleepDur)
	var n uint32
	mux := func(reqCtx *fasthttp.RequestCtx) {
		atomic.AddUint32(&n, 1)
		reqCtx.Write([]byte(expectedBodyStr))
	}

	cachedMux := httpcache.RequestFasthttp(remotescheme+fasthttpremoteaddr, mux, cacheDuration)
	e := httptest.New(t, httptest.RequestHandler(cachedMux))

	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}
}
*/