package httpcache

import (
	"net/http"
	"time"

	"github.com/geekypanda/httpcache/internal/fhttp"
	"github.com/geekypanda/httpcache/internal/nethttp"
	"github.com/geekypanda/httpcache/internal/server"
	"github.com/valyala/fasthttp"
)

const (
	// Version is the release version number of the httpcache package.
	Version = "0.0.2"
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
	return nethttp.NewHandler(bodyHandler, expiration)
}

// CacheFunc accepts two parameters
// first is the http.HandlerFunc which you want to cache its result
// the second is, optional, the cache Entry's expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns an http.HandlerFunc, which you can use as your default router or per-route handler
//
// All type of responses are cached, templates, json, text, anything.
//
// If you use only one global cache for all of your routes use the `httpcache.New` instead
func CacheFunc(bodyHandler func(http.ResponseWriter, *http.Request), expiration time.Duration) http.HandlerFunc {
	return http.HandlerFunc(nethttp.NewHandler(http.HandlerFunc(bodyHandler), expiration).ServeHTTP)
}

// CacheFasthttp accepts two parameters
// first is the fasthttp.RequestHandler which you want to cache its result
// the second is, optional, the cache Entry's expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns an http.Handler, which you can use as your default router or per-route handler
//
// All type of responses are cached, templates, json, text, anything.
//
// If you use only one global cache for all of your routes use the `httpcache.New` instead
func CacheFasthttp(bodyHandler fasthttp.RequestHandler, expiration time.Duration) fasthttp.RequestHandler {
	return fhttp.NewHandler(bodyHandler, expiration).ServeHTTP
}

// distributed

// ListenAndServe receives a network address and starts a server
// with a remote server cache handler registered to it
// which handles remote client-side cache handlers
// client should register its handlers with the RemoteCache & RemoteCacheFasthttp functions
//
// Note: It doesn't starts the server,
func ListenAndServe(addr string) error {
	return server.New(addr, nil).ListenAndServe()
}

// CacheRemote receives a handler, its cache expiration and
// the remote address of the remote cache server(look ListenAndServe)
// returns a remote-cached handler
func CacheRemote(bodyHandler http.Handler, expiration time.Duration, remoteServerAddr string) http.Handler {
	return nethttp.NewClientHandler(bodyHandler, expiration, remoteServerAddr)
}

// CacheRemoteFunc receives a handler function, its cache expiration and
// the remote address of the remote cache server(look ListenAndServe)
// returns a remote-cached handler function
func CacheRemoteFunc(bodyHandler func(http.ResponseWriter, *http.Request), expiration time.Duration, remoteServerAddr string) http.HandlerFunc {
	return http.HandlerFunc(nethttp.NewClientHandler(http.HandlerFunc(bodyHandler), expiration, remoteServerAddr).ServeHTTP)
}

// CacheRemoteFasthttp receives a fasthttp handler, its cache expiration and
// the remote address of the remote cache server(look ListenAndServe)
// returns a remote-cached handler
func CacheRemoteFasthttp(bodyHandler fasthttp.RequestHandler, expiration time.Duration, remoteServerAddr string) fasthttp.RequestHandler {
	return fhttp.NewClientHandler(bodyHandler, expiration, remoteServerAddr).ServeHTTP
}
