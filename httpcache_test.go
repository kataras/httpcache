package httpcache_test

import (
	"github.com/gavv/httpexpect"
	"github.com/geekypanda/httpcache"
	"github.com/geekypanda/httpcache/httptest"
	"github.com/kataras/go-errors"
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
	cacheDuration      = 5 * time.Second
	serverSleepDur     = 3 * time.Second
	expectedBodyStr    = "Imagine it as a big message to achieve x20 response performance!"
	errTestFailed      = errors.New("Expected the main handler to be executed %d times instead of %d.")
)

// ~14secs
func runTest(e *httpexpect.Expect, counterPtr *uint32, expectedBodyStr string) error {
	e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	time.Sleep(cacheDuration / 5) // lets wait for a while, cache should be saved and ready
	e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	counter := atomic.LoadUint32(counterPtr)
	if counter > 1 {
		// n should be 1 because it doesn't changed after the first call
		return errTestFailed.Format(1, counter)
	}
	time.Sleep(cacheDuration)

	// cache should be cleared now
	e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	time.Sleep(cacheDuration / 5)
	// let's call again , the cache should be saved
	e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	counter = atomic.LoadUint32(counterPtr)
	if counter != 2 {
		return errTestFailed.Format(2, counter)
	}

	// we have cache response saved for the "/" path, we have some time more here, but here
	// we will make the requestS with one of the denier branch, let's take the "maxage=0"
	// the ORIGINAL HANDLER SHOULD BE EXECUTED, NOT THE CACHED, so the counter will be ++
	e.GET("/").WithHeader("max-age", "0").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	e.GET("/").WithHeader("Authorization", "basic or anything").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	counter = atomic.LoadUint32(counterPtr)
	if counter != 4 {
		return errTestFailed.Format(4, counter)
	}

	return nil
}

func TestCache(t *testing.T) {
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

func TestCacheFasthttp(t *testing.T) {
	var n uint32
	mux := func(reqCtx *fasthttp.RequestCtx) {
		atomic.AddUint32(&n, 1)
		reqCtx.Write([]byte(expectedBodyStr))
	}

	cachedMux := httpcache.CacheFasthttp(mux, cacheDuration)
	e := httptest.New(t, httptest.RequestHandler(cachedMux))
	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}
}

func TestCacheDistributed(t *testing.T) {
	// start the remote cache service
	go httpcache.ListenAndServe(httpremoteaddr)
	time.Sleep(serverSleepDur) // let's wait a little

	// make the client
	mux := http.NewServeMux()

	var n uint32

	mux.Handle("/", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddUint32(&n, 1)
		res.Write([]byte(expectedBodyStr))
	}))

	cachedMux := httpcache.CacheRemote(mux, cacheDuration, remotescheme+httpremoteaddr)
	e := httptest.New(t, httptest.Handler(cachedMux))
	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}
}

func TestCacheDistributedFasthttp(t *testing.T) {
	// start the remote cache service
	go httpcache.ListenAndServe(fasthttpremoteaddr)
	time.Sleep(serverSleepDur) // let's wait a little
	var n uint32
	mux := func(reqCtx *fasthttp.RequestCtx) {
		atomic.AddUint32(&n, 1)
		reqCtx.Write([]byte(expectedBodyStr))
	}

	cachedMux := httpcache.CacheRemoteFasthttp(mux, cacheDuration, remotescheme+fasthttpremoteaddr)
	e := httptest.New(t, httptest.RequestHandler(cachedMux))

	if err := runTest(e, &n, expectedBodyStr); err != nil {
		t.Fatal(err)
	}
}

func TestCacheParallel(t *testing.T) {
	t.Parallel()
	TestCache(t)
}

func TestCacheFasthttpParallel(t *testing.T) {
	t.Parallel()
	TestCacheFasthttp(t)
}

func TestCacheDistributedParallel(t *testing.T) {
	t.Parallel()
	TestCacheDistributed(t)
}

func TestCacheDistributedFasthttpParallel(t *testing.T) {
	t.Parallel()
	TestCacheDistributedFasthttp(t)
}

//
