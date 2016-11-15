package nethttp

import (
	"bytes"
	"github.com/geekypanda/httpcache/internal"
	"io/ioutil"
	"net/http"
	"time"
)

// ClientHandler is the client-side handler
// for each of the cached route paths's response
// register one client handler per route.
//
// it's just calls a remote cache service server/handler,
//  which lives on other, external machine.
//
type ClientHandler struct {
	// bodyHandler the original route's handler
	bodyHandler http.Handler

	life time.Duration

	remoteHandlerURL string
}

// NewClientHandler returns a new remote client handler
// which asks the remote handler the cached entry's response
// with a GET request, or add a response with POST request
// these all are done automatically, users can use this
// handler as they use the local.go/NewHandler
//
// the ClientHandler is useful when user
// wants to apply horizontal scaling to the app and
// has a central http server which handles
func NewClientHandler(bodyHandler http.Handler, life time.Duration, remote string) *ClientHandler {
	return &ClientHandler{
		bodyHandler:      bodyHandler,
		life:             life,
		remoteHandlerURL: remote,
	}
}

// Client is used inside the global Request function
// this client is an exported to give you a freedom of change its Transport, Timeout and so on(in case of ssl)
var Client = &http.Client{Timeout: internal.RequestCacheTimeout}

const (
	methodGet  = "GET"
	methodPost = "POST"
)

// ServeHTTP , or remote cache client whatever you like, it's the client-side function of the ServeHTTP
// sends a request to the server-side remote cache Service and sends the cached response to the frontend client
// it is used only when you achieved something like horizontal scaling (separate machines)
// look ../remote/remote.ServeHTTP for more
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
func (h *ClientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uri := &internal.URIBuilder{}
	uri.ServerAddr(h.remoteHandlerURL).ClientURI(r.URL.RequestURI()).ClientMethod(r.Method)

	// set the full url here because below we have other issues, probably net/http bugs
	request, err := http.NewRequest(methodGet, uri.String(), nil)
	if err != nil {
		//// println("error when requesting to the remote service: " + err.Error())
		// somehing very bad happens, just execute the user's handler and return
		h.bodyHandler.ServeHTTP(w, r)
		return
	}

	// println("GET Do to the remote cache service with the url: " + request.URL.String())
	response, err := Client.Do(request)

	if err != nil || response.StatusCode == internal.FailStatus {
		// if not found on cache, then execute the handler and save the cache to the remote server
		recorder := AcquireResponseRecorder(w)
		h.bodyHandler.ServeHTTP(recorder, r)
		// save to the remote cache
		// we re-create the request for any case

		body := recorder.Body()[0:]
		if len(body) == 0 {
			//// println("Request: len body is zero, do nothing")
			return
		}
		uri.StatusCode(recorder.StatusCode())
		uri.Lifetime(h.life)
		uri.ContentType(recorder.ContentType())

		request, err = http.NewRequest(methodPost, uri.String(), bytes.NewBuffer(body)) // yes new buffer every time
		ReleaseResponseRecorder(recorder)
		// println("POST Do to the remote cache service with the url: " + request.URL.String())
		if err != nil {
			//// println("Request: error on method Post of request to the remote: " + err.Error())
			return
		}
		// go Client.Do(request)
		Client.Do(request)
	} else {
		// get the status code , content type and the write the response body
		w.Header().Set(internal.ContentTypeHeader, response.Header.Get(internal.ContentTypeHeader))
		w.WriteHeader(response.StatusCode)
		responseBody, err := ioutil.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			return
		}
		w.Write(responseBody)

	}
}
