package main

import (
	"net/http"
	"time"

	"github.com/geekypanda/httpcache"
	"github.com/kataras/go-template"
	"github.com/kataras/go-template/html"
)

// In this example we will see how custom templates are cached,
// the same code snippet (httpcache.Cache/httpcache.CacheFunc) is working for everything else.

// Here we're using a custom package which handles the templates with ease,
//  you can use the standard way too.
func init() {
	e := html.New(html.Config{Layout: "layouts/layout.html"})
	template.AddEngine(e)
	if err := template.Load(); err != nil {
		panic(err)
	}
}

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

type mypage struct {
	Title   string
	Message string
}

func mypageHandler(w http.ResponseWriter, r *http.Request) {
	// tap multiple times the browser's refresh button and you will
	// see this println only once each of 20seconds
	println("Handler executed. Cache refreshed.")

	// set our content type and send the response to the client, it can be any type of response
	// we just select templates to show you something 'real'
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	template.ExecuteWriter(w, "mypage.html", mypage{"My Page title", "Hello world!"})
}
