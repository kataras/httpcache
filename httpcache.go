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
	"net/http"
	"time"
)

var (
	// GC checks for expired cache entries and deletes them every tick of this time
	// set it to <=2*time.Second to disable GC for all default cache services'  memory stores
	GC = 30 * time.Minute

	// defaultStore is the default memory store for the cache services,
	// net/http and fasthttp cache  services are sharing the same store for easly scaling by-default
	defaultStore = NewMemoryStore(GC)

	// Fasthttp is the package-level instance of valyala/fasthttp cache service with a memory store
	// has the same functions as httpcache and httpcache.NewService(), which is the net/http cache
	//
	// USAGE:
	// fasthttp.ListenAndServe("mydomain.com",httpcache.Fasthttp.Cache(myRouter, 20* time.Second))
	// or for individual handlers:
	// httpcache.Fasthttp.Cache(myRequestHandler, 20* time.Secnd)
	//
	Fasthttp = NewServiceFasthttp(defaultStore)

	// Default is default package-level instance of the net/http cache service with a memory store
	//
	// USAGE:
	// http.ListenAndServe("mydomain.com",httpcache.Cache(myRouter, 20* time.Second))
	// or for individual handlers:
	// httpcache.Cache(myHandler, 20* time.Secnd)
	//
	Default = NewService(defaultStore)
)

// Cache accepts two parameters
// first is the http.Handler which you want to cache its result
// the second is, optional, the cache Entry's expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns an http.Handler, which you can use as your default router or per-route handler
//
// All type of responses are cached, templates, json, text, anything.
//
// If you use only one global cache for all of your routes use the `httpcache.New` instead
func Cache(bodyHandler http.Handler, expiration time.Duration) http.Handler {
	return Default.Cache(bodyHandler, expiration)
}

// Invalidate accepts a *http.Request which is used to find the cache key
// and removes any releated Entry from the cache
func Invalidate(req *http.Request) {
	Default.Invalidate(req)
}

// ServeHTTP serves the cache Service to the outside world,
// it is used only when you want to achieve something like horizontal scaling (separate machines)
// it parses the request and tries to return the response with the cached body of the requested cache key
// server-side function
func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	Default.ServeHTTP(res, req)
}

// Clear removes all entries from the default cache
func Clear() {
	defaultStore.Clear()
}
