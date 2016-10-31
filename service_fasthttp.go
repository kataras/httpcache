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
	"github.com/valyala/fasthttp"
	"net/url"
	"strconv"
	"time"
)

func getResponseStatusCodeFasthttp(reqCtx *fasthttp.RequestCtx) int {
	return validateStatusCode(reqCtx.Response.StatusCode())
}

func getResponseContentTypeFasthttp(reqCtx *fasthttp.RequestCtx) string {
	return validateContentType(string(reqCtx.Response.Header.ContentType()))
}

func getMaxAgeFasthttp(reqCtx *fasthttp.RequestCtx) int64 {
	header := string(reqCtx.Request.Header.Peek(cacheControlHeader))
	return parseMaxAge(header)
}

// getCacheKeyFasthttp returns the cache key(string) from a fasthttp.RequestCtx
// it's just the RequestURI (path+ escaped query)
func getCacheKeyFasthttp(reqCtx *fasthttp.RequestCtx) string {
	return string(reqCtx.URI().RequestURI())
}

// remote

// parseClientRequestURI returns the full url which should be passed to get a cache entry response back
// (it could be setted by server too but we need some client-freedom on the requested key)
// in order to be sure that the registered cache entries are unique among different clients with the same key
func parseClientRequestURIFasthttp(remoteStoreCacheURL string, reqCtx *fasthttp.RequestCtx) string {
	return remoteStoreCacheURL + "?" + queryCacheKey + "=" + url.QueryEscape(string(reqCtx.Method())+string(reqCtx.URI().Scheme())+string(reqCtx.URI().Host())+getCacheKeyFasthttp(reqCtx))
}

func getURLParamFasthttp(reqCtx *fasthttp.RequestCtx, key string) string {
	return string(reqCtx.Request.URI().QueryArgs().Peek(key))
}

func getURLParamIntFasthttp(reqCtx *fasthttp.RequestCtx, key string) (int, error) {
	return strconv.Atoi(getURLParamFasthttp(reqCtx, key))
}

func getURLParamInt64Fasthttp(reqCtx *fasthttp.RequestCtx, key string) (int64, error) {
	return strconv.ParseInt(getURLParamFasthttp(reqCtx, key), 10, 64)
}

// ServiceFasthttp contains the cache and performs actions to save cache and serve cached content
type ServiceFasthttp struct {
	store Store
}

// NewServiceFasthttp returns a new cache Service with the given store
func NewServiceFasthttp(store Store) *ServiceFasthttp {
	return &ServiceFasthttp{store: store}
}

// Invalidate accepts a *http.Request which is used to find the cache key
// and removes any releated Entry from the cache
func (s *ServiceFasthttp) Invalidate(reqCtx *fasthttp.RequestCtx) {
	key := getCacheKeyFasthttp(reqCtx)
	s.store.Remove(key)
}

// Clear clears all the cache (the GC still running)
func (s *ServiceFasthttp) Clear() {
	s.store.Clear()
}

// Cache accepts two parameters
// first is the http.Handler which you want to cache its result
// the second is, optional, the cache Entry's expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns an http.Handler, which you can use as your default router or per-route handler
//
// All type of responses are cached, templates, json, text, anything.
//
// If you use only one global cache for all of your routes use the `httpcache.New` instead
func (s *ServiceFasthttp) Cache(bodyHandler fasthttp.RequestHandler, expiration time.Duration) fasthttp.RequestHandler {
	expiration = validateCacheDuration(expiration)

	h := func(reqCtx *fasthttp.RequestCtx) {
		key := getCacheKeyFasthttp(reqCtx)
		if v := s.store.Get(key); v != nil {
			reqCtx.Response.Header.Set(contentTypeHeader, v.ContentType)
			reqCtx.SetStatusCode(v.StatusCode)
			reqCtx.Write(v.Body)
			return
		}

		// if not found then serve the handler and collect its results after
		bodyHandler(reqCtx)
		// and set the cache value as its response body in a goroutine, because we want to exit from the route's handler as soon as possible
		body := reqCtx.Response.Body()[0:]

		if len(body) == 0 {
			return
		}

		if expiration <= minimumAllowedCacheDuration {
			// try to set the expiraion from header
			expiration = time.Duration(getMaxAgeFasthttp(reqCtx)) * time.Second
		}

		cType := getResponseContentTypeFasthttp(reqCtx)
		statusCode := reqCtx.Response.StatusCode()

		go s.store.Set(key, statusCode, cType, body, expiration)

	}

	return h
}

// ServeHTTP serves the cache ServiceFasthttp to the outside world,
// it is used only when you want to achieve something like horizontal scaling
// it parses the request and tries to return the response with the cached body of the requested cache key
// server-side function
func (s *ServiceFasthttp) ServeHTTP(reqCtx *fasthttp.RequestCtx) {
	key := getURLParamFasthttp(reqCtx, queryCacheKey)
	if key == "" {
		reqCtx.SetStatusCode(failStatus)
		return
	}

	if reqCtx.IsGet() {
		// println("[FASTHTTP] ServeHTTP: GET key = " + key)
		if v := s.store.Get(key); v != nil {
			reqCtx.Response.Header.Set(contentTypeHeader, v.ContentType)
			reqCtx.SetStatusCode(v.StatusCode)
			reqCtx.Write(v.Body)
			// println("[FASTHTTP] ServeHTTP:  found and sent")
			return
		}
		// println("[FASTHTTP] ServeHTTP: not found ")
		reqCtx.SetStatusCode(failStatus)
		return
	} else if reqCtx.IsPost() {
		// println("[FASTHTTP] ServeHTTP: POST save a cache entry ")
		expirationSeconds, err := getURLParamInt64Fasthttp(reqCtx, queryCacheDuration)
		// get the body from the requested body
		// get the expiration from the "cache-control's maxage" if no url param is setted
		if expirationSeconds <= 0 || err != nil {
			expirationSeconds = getMaxAgeFasthttp(reqCtx)
		}
		// if not setted then try to get it via
		if expirationSeconds <= 0 {
			expirationSeconds = int64(minimumAllowedCacheDuration.Seconds())
		}
		// get the body from the requested body
		body := reqCtx.Request.Body()[0:]
		if len(body) == 0 {
			// println("[FASTHTTP] ServeHTTP: request's body was empty, return fail! ")
			reqCtx.SetStatusCode(failStatus)
			return

		}

		cacheDuration := validateCacheDuration(time.Duration(expirationSeconds) * time.Second)
		statusCode, _ := getURLParamIntFasthttp(reqCtx, queryCacheStatusCode)
		statusCode = validateStatusCode(statusCode)
		cType := validateContentType(getURLParamFasthttp(reqCtx, queryCacheContentType))

		// println("[FASTHTTP] ServeHTTP: set into store: key =  "+key+" content-type: "+cType+" expiration : ", int(cacheDuration.Seconds()))
		s.store.Set(key, statusCode, cType, body, cacheDuration)

		reqCtx.SetStatusCode(successStatus)
		return
	} else if reqCtx.IsDelete() {
		s.store.Remove(key)
		reqCtx.SetStatusCode(successStatus)
		return
	}

	reqCtx.SetStatusCode(failStatus)
}

// ClientFasthttp is used inside the global RequestFasthttp function
// this client is an exported variable because the maybe the remote cache service is running behind ssl,
// in that case you are able to set a Transport inside it
var ClientFasthttp = &fasthttp.Client{WriteTimeout: requestCacheTimeout, ReadTimeout: requestCacheTimeout}

// RequestFasthttp , it's the client-side function, sends a request to the server-side remote cache Service and parses the cached response
// it is used only when you achieved something like horizontal scaling (separate machines) and you have an already running remote cache Service or ServiceFasthttp
// look ServeHTTP for more
//
// if cache din't find then it sends a POST request and save the bodyHandler's body to the remote cache.
//
// It takes 3 parameters
// the first is the remote address (it's the address you started your http server which handled by the Service.ServeHTTP)
// the second is the handler (or the mux) you want to cache
// and the  third is the, optionally recommending, cache expiration
// which is used to set cache duration of this specific cache entry to the remote cache service
//
// client-side function
func RequestFasthttp(remoteURL string, bodyHandler fasthttp.RequestHandler, expiration time.Duration) fasthttp.RequestHandler {
	cacheDurationStr := strconv.Itoa(int(expiration.Seconds()))

	return func(reqCtx *fasthttp.RequestCtx) {
		uri := remoteURI{
			remoteURL: remoteURL,
			key:       getCacheKeyFasthttp(reqCtx),
			method:    string(reqCtx.Method()),
			scheme:    string(reqCtx.URI().Scheme()),
			host:      string(reqCtx.URI().Host()),
		}

		req := fasthttp.AcquireRequest()

		req.URI().Update(uri.String())
		req.Header.SetMethodBytes(methodGetBytes)

		res := fasthttp.AcquireResponse()
		// println("[FASTHTTP] GET Do to the remote cache service with the url: " + req.URI().String())

		err := ClientFasthttp.Do(req, res)
		if err != nil || res.StatusCode() == failStatus {
			// if not found on cache, then execute the handler and save the cache to the remote server
			bodyHandler(reqCtx)
			// save to the remote cache

			body := reqCtx.Response.Body()[0:]
			if len(body) == 0 {
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(res)
				return // do nothing..
			}
			req.Reset()

			uri.statusCodeStr = strconv.Itoa(reqCtx.Response.StatusCode())
			uri.cacheDurationStr = cacheDurationStr
			uri.contentType = validateContentType(string(reqCtx.Response.Header.Peek(contentTypeHeader)))

			req.URI().Update(uri.String())
			req.Header.SetMethodBytes(methodPostBytes)
			req.SetBody(body)

			//	go func() {
			// println("[FASTHTTP] POST Do to the remote cache service with the url: " + req.URI().String() + " , method validation: " + string(req.Header.Method()))
			err := ClientFasthttp.Do(req, res)
			if err != nil {
				// println("[FASTHTTP] ERROR WHEN POSTING TO SAVE THE CACHE ENTRY. TRACE: " + err.Error())
			}
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(res)
			//	}()

		} else {
			// get the status code , content type and the write the response body
			statusCode := res.StatusCode()
			// println("[FASTHTTP] ServeHTTP: WRITE WITH THE CACHED, StatusCode: ", statusCode)
			cType := res.Header.ContentType()
			reqCtx.SetStatusCode(statusCode)
			reqCtx.Response.Header.SetContentTypeBytes(cType)

			reqCtx.Write(res.Body())

			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(res)
		}

	}
}
