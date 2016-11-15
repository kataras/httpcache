package main

import (
	"net/http"
	"time"

	"github.com/geekypanda/httpcache"
)

func main() {
	// The only thing that separates your handler to be cached is just ONE function wrapper
	// httpcache.CacheFunc will cache your http.HandlerFunc
	// httpcache.Cache will cache your http.Handler
	//
	// first argument is the handler witch serves the contents to the client
	// second argument is how long this cache will be valid
	// and must be refreshed after the x time passed and a new request comes
	http.HandleFunc("/", httpcache.CacheFunc(mypageHandler, 20*time.Second))

	// start the server, navigate to http://localhost:8080
	http.ListenAndServe(":8080", nil)
}

func mypageHandler(w http.ResponseWriter, r *http.Request) {
	// tap multiple times the browser's refresh button and you will
	// see this println only once each of 20seconds
	println("Handler executed. Cache refreshed.")

	// set our content type and send the response to the client,
	// it can be any type of response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1> Hello!!! </h1>"))
}
