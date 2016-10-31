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
	"bytes"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func getResponseStatusCode(res http.ResponseWriter) int {
	statusCodeStr := res.Header().Get(statusCodeHeader)
	statusCode, _ := strconv.Atoi(statusCodeStr)
	return validateStatusCode(statusCode)
}

func getResponseContentType(res http.ResponseWriter) string {
	return validateContentType(res.Header().Get(contentTypeHeader))
}

func getMaxAge(req *http.Request) int64 {
	header := req.Header.Get(cacheControlHeader)
	return parseMaxAge(header)
}

// getCacheKey returns the cache key(string) from an http.Request
// it's just the path+escaped query
func getCacheKey(req *http.Request) string {
	return req.URL.EscapedPath()
}

// remote

func getURLParam(req *http.Request, key string) string {
	return req.URL.Query().Get(key)
}

func getURLParamInt(req *http.Request, key string) (int, error) {
	return strconv.Atoi(getURLParam(req, key))
}

func getURLParamInt64(req *http.Request, key string) (int64, error) {
	return strconv.ParseInt(getURLParam(req, key), 10, 64)
}

// Service contains the cache and performs actions to save cache and serve cached content
type Service struct {
	store Store
}

// NewService returns a new cache Service with the given store
func NewService(store Store) *Service {
	return &Service{store: store}
}

/*NOTE: We could use the same Service for both net/http and fasthttp with the 'functional options pattern' but
Go has lack on generics and the return type should be statically written,
so... we stick with two service with the same exactly structure...
the developer still be able to use the same store for more than one service in the same app*
*/

// Cache accepts two parameters
// first is the http.Handler which you want to cache its result
// the second is, optional, the cache Entry's expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns an http.Handler, which you can use as your default router or per-route handler
//
// All type of responses are cached, templates, json, text, anything.
//
// If you use only one global cache for all of your routes use the `httpcache.New` instead
func (s *Service) Cache(bodyHandler http.Handler, expiration time.Duration) http.Handler {
	expiration = validateCacheDuration(expiration)

	h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		key := getCacheKey(req)
		if v := s.store.Get(key); v != nil {
			res.Header().Set(contentTypeHeader, v.ContentType)
			res.WriteHeader(v.StatusCode)
			res.Write(v.Body)
			return
		}

		recorder := acquireResponseRecorder(res)
		// if not found then serve the handler and collect its results after
		bodyHandler.ServeHTTP(recorder, req)

		body := recorder.getBody()[0:]

		if len(body) == 0 {
			return
		}

		if expiration <= minimumAllowedCacheDuration {
			// try to set the expiraion from header
			expiration = time.Duration(getMaxAge(req)) * time.Second
		}

		cType := getResponseContentType(recorder)
		statusCode := recorder.statusCode
		releaseResponseRecorder(recorder)

		// and set the cache value as its response body in a goroutine, because we want to exit from the route's handler as soon as possible
		go s.store.Set(key, statusCode, cType, body, expiration)
	})

	return h
}

// Invalidate accepts a *http.Request which is used to find the cache key
// and removes any releated entry from the cache
func (s *Service) Invalidate(req *http.Request) {
	key := getCacheKey(req)
	s.store.Remove(key)
}

// Clear removes all cache entries(the GC still running)
func (s *Service) Clear() {
	s.store.Clear()
}

// ServeHTTP serves the cache Service to the outside world,
// it is used only when you want to achieve something like horizontal scaling
// it parses the request and tries to return the response with the cached body of the requested cache key
// server-side function
func (s *Service) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// println("Request to the remote service has been established")
	key := getURLParam(req, queryCacheKey)
	if key == "" {
		// println("return because key was empty")
		res.WriteHeader(failStatus)
		return
	}

	// println("ServeHTTP: key=" + key)

	if req.Method == methodGet {
		// println("we have a get request method, let's give back the cached entry ")
		if v := s.store.Get(key); v != nil {
			res.Header().Set(contentTypeHeader, v.ContentType)
			res.WriteHeader(v.StatusCode)
			res.Write(v.Body)
			// println("cached entry sent")
			return
		}
		// println("ServeHTTP: Cache didn't found")
		res.WriteHeader(failStatus)
		return
	} else if req.Method == methodPost {
		// println("we have a post request method, let's save a cached entry ")
		// get the cache expiration via url param
		expirationSeconds, err := getURLParamInt64(req, queryCacheDuration)
		// get the body from the requested body
		// get the expiration from the "cache-control's maxage" if no url param is setted
		if expirationSeconds <= 0 || err != nil {
			expirationSeconds = getMaxAge(req)
		}
		// if not setted then try to get it via
		if expirationSeconds <= 0 {
			expirationSeconds = int64(minimumAllowedCacheDuration.Seconds())
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil || len(body) == 0 {
			// println("body's request was empty, return fail")
			res.WriteHeader(failStatus)
			return
		}

		cacheDuration := validateCacheDuration(time.Duration(expirationSeconds) * time.Second)
		statusCode, _ := getURLParamInt(req, queryCacheStatusCode)
		statusCode = validateStatusCode(statusCode)
		cType := validateContentType(getURLParam(req, queryCacheContentType))

		// store by its url+the key in order to be unique key among different servers with the same paths
		s.store.Set(key, statusCode, cType, body, cacheDuration)
		res.WriteHeader(successStatus)
		// println("ok, save the cache from the request")
		return
	} else if req.Method == methodDelete {
		s.store.Remove(key)
		res.WriteHeader(successStatus)
		// println("requested with delete, let's invalidate the cache key = " + key)
		return
	}
	// println("no get no post no delete method, return fail status!!")
	res.WriteHeader(failStatus)
}

// Client is used inside the global Request function
// this client is an exported to give you a freedom of change its Transport, Timeout and so on(in case of ssl)
var Client = &http.Client{Timeout: requestCacheTimeout}

// Request , or remote cache client whatever you like, it's the client-side function of the ServeHTTP
// sends a request to the server-side remote cache Service and sends the cached response to the frontend client
// it is used only when you achieved something like horizontal scaling (separate machines)
// look ServeHTTP for more
//
// if cache din't find then it sends a POST request and save the bodyHandler's body to the remote cache.
//
// It takes 3 parameters
// the first is the remote address (it's the address you started your http server which handled by the Service.ServeHTTP)
// the second is the handler (or the mux) you want to cache
// and the  third is the, optionally, cache expiration,
// which is used to set cache duration of this specific cache entry to the remote cache service
// if <=minimumAllowedCacheDuration then the server will try to parse from "cache-control" header
//
// client-side function
func Request(remoteURL string, bodyHandler http.Handler, expiration time.Duration) http.Handler {
	cacheDurationStr := strconv.Itoa(int(expiration.Seconds()))
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		uri := remoteURI{
			remoteURL: remoteURL,
			key:       getCacheKey(req),
			method:    req.Method,
			scheme:    req.URL.Scheme,
			host:      req.Host,
		}
		// set the full url here because below we have other issues, probably net/http bugs
		request, err := http.NewRequest(methodGet, uri.String(), nil)
		if err != nil {
			//// println("error when requesting to the remote service: " + err.Error())
			// somehing very bad happens, just execute the user's handler and return
			bodyHandler.ServeHTTP(res, req)
			return
		}

		// println("GET Do to the remote cache service with the url: " + request.URL.String())
		response, err := Client.Do(request)

		if err != nil || response.StatusCode == failStatus {
			// if not found on cache, then execute the handler and save the cache to the remote server
			recorder := acquireResponseRecorder(res)
			bodyHandler.ServeHTTP(recorder, req)
			// save to the remote cache
			// we re-create the request for any case

			body := recorder.getBody()[0:]
			if len(body) == 0 {
				//// println("Request: len body is zero, do nothing")
				return
			}
			uri.statusCodeStr = strconv.Itoa(recorder.statusCode)
			uri.cacheDurationStr = cacheDurationStr
			uri.contentType = validateContentType(recorder.Header().Get(contentTypeHeader))
			releaseResponseRecorder(recorder)
			request, err = http.NewRequest(methodPost, uri.String(), bytes.NewBuffer(body)) // yes new buffer everytime

			// println("POST Do to the remote cache service with the url: " + request.URL.String())
			if err != nil {
				//// println("Request: error on method Post of request to the remote: " + err.Error())
				return
			}
			// go Client.Do(request)
			Client.Do(request)
		} else {
			// get the status code , content type and the write the response body
			res.Header().Set(contentTypeHeader, response.Header.Get(contentTypeHeader))
			res.WriteHeader(response.StatusCode)
			responseBody, err := ioutil.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				return
			}
			res.Write(responseBody)

		}

	})
}
