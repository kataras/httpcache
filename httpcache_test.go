package httpcache_test

import (
	"github.com/gavv/httpexpect"
	"github.com/geekypanda/httpcache"
	"github.com/geekypanda/httpcache/httptest"
	"github.com/geekypanda/httpcache/internal/nethttp/rule"
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
func runTest(e *httpexpect.Expect, counterPtr *uint32, expectedBodyStr string, nocache string) error {
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
	// we will make the requestS with some of the deniers options
	e.GET("/").WithHeader("max-age", "0").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	e.GET("/").WithHeader("Authorization", "basic or anything").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	counter = atomic.LoadUint32(counterPtr)
	if counter != 4 {
		return errTestFailed.Format(4, counter)
	}

	if nocache != "" {
		// test the NoCache, first sleep to pass the cache expiration,
		// second add to the cache with a valid request and response
		// third, do it with the "/nocache" path (static for now, pure test design) given by the consumer
		time.Sleep(cacheDuration)

		// cache should be cleared now, this should work because we are not in the "nocache" path
		e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr) // counter = 5
		time.Sleep(cacheDuration / 5)

		// let's call the "nocache", the expiration is not passed so but the "nocache"
		// route's path has the httpcache.NoCache so it should be not cached and the counter should be ++
		e.GET(nocache).Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr) // counter should be 6
		counter = atomic.LoadUint32(counterPtr)
		if counter != 6 { // 4 before, 5 with the first call to store the cache, and six with the no cache, again original handler executation
			return errTestFailed.Format(6, counter)
		}

		// let's call again the "/", the expiration is not passed so  it should be cached
		e.GET("/").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
		counter = atomic.LoadUint32(counterPtr)
		if counter != 6 {
			return errTestFailed.Format(6, counter)
		}

		// but now check for the No
	}

	return nil
}

func TestNoCache(t *testing.T) {
	mux := http.NewServeMux()
	var n uint32

	mux.Handle("/", httpcache.CacheFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddUint32(&n, 1)
		res.Write([]byte(expectedBodyStr))
	}, cacheDuration))

	mux.Handle("/nocache", httpcache.CacheFunc(func(res http.ResponseWriter, req *http.Request) {
		httpcache.NoCache(res) // <----

		atomic.AddUint32(&n, 1)
		res.Write([]byte(expectedBodyStr))
	}, cacheDuration))

	e := httptest.New(t, httptest.Handler(mux))
	if err := runTest(e, &n, expectedBodyStr, "/nocache"); err != nil {
		t.Fatal(err)
	}

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
	if err := runTest(e, &n, expectedBodyStr, ""); err != nil {
		t.Fatal(err)
	}

}

func TestCacheFasthttp(t *testing.T) {
	var n uint32
	mux := func(reqCtx *fasthttp.RequestCtx) {
		atomic.AddUint32(&n, 1)
		reqCtx.Write([]byte(expectedBodyStr))
	}

	cachedMux := httpcache.CacheFasthttpFunc(mux, cacheDuration)
	e := httptest.New(t, httptest.RequestHandler(cachedMux))
	if err := runTest(e, &n, expectedBodyStr, ""); err != nil {
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
	if err := runTest(e, &n, expectedBodyStr, ""); err != nil {
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

	cachedMux := httpcache.CacheRemoteFasthttpFunc(mux, cacheDuration, remotescheme+fasthttpremoteaddr)
	e := httptest.New(t, httptest.RequestHandler(cachedMux))

	if err := runTest(e, &n, expectedBodyStr, ""); err != nil {
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

func TestCacheValidator(t *testing.T) {
	mux := http.NewServeMux()
	var n uint32

	h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddUint32(&n, 1)
		res.Write([]byte(expectedBodyStr))
	})

	validCache := httpcache.Cache(h, cacheDuration)
	mux.Handle("/", validCache)

	managedCache := httpcache.Cache(h, cacheDuration)
	managedCache.AddRule(rule.Validator([]rule.PreValidator{
		func(r *http.Request) bool {
			if r.URL.Path == "/invalid" {
				return false // should always invalid for cache, don't bother to go to try to get or set cache
			}
			return true
		},
	}, nil))

	managedCache2 := httpcache.Cache(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddUint32(&n, 1)
		res.Header().Set("DONT", "DO not cache that response even if it was claimed")
		res.Write([]byte(expectedBodyStr))

	}), cacheDuration)
	managedCache2.AddRule(rule.Validator(nil,
		[]rule.PostValidator{
			func(w http.ResponseWriter, r *http.Request) bool {
				if w.Header().Get("DONT") != "" {
					return false // it's passed the Claim and now Valid checks if the response contains a header of "DONT"
				}
				return true
			},
		},
	))

	mux.Handle("/valid", validCache)

	mux.Handle("/invalid", managedCache)
	mux.Handle("/invalid2", managedCache2)

	e := httptest.New(t, httptest.Handler(mux))

	// execute from cache the next time
	e.GET("/valid").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	time.Sleep(cacheDuration / 5) // lets wait for a while, cache should be saved and ready
	e.GET("/valid").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)
	counter := atomic.LoadUint32(&n)
	if counter > 1 {
		// n should be 1 because it doesn't changed after the first call
		t.Fatal(errTestFailed.Format(1, counter))
	}
	// don't execute from cache, execute the original, counter should ++ here
	e.GET("/invalid").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr)  // counter = 2
	e.GET("/invalid2").Expect().Status(http.StatusOK).Body().Equal(expectedBodyStr) // counter = 3

	counter = atomic.LoadUint32(&n)
	if counter != 3 {
		// n should be 1 because it doesn't changed after the first call
		t.Fatal(errTestFailed.Format(3, counter))
	}
}
