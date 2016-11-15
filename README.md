httpcache is an automation web cache service written in Go.

Compatible with [net/http](https://golang.org/pkg/net/http/) and [valyala/fasthttp](https://github.com/valyala/fasthttp).

[![Travis Widget]][Travis] [![Release Widget]][Release] [![Report Widget]][Report] [![License Widget]][License] [![Chat Widget]][Chat]


A web cache (or HTTP cache) is an information technology for the
temporary storage (caching) of web documents,
such as HTML pages and images, to reduce bandwidth usage,
server load, and perceived lag. A web cache system stores
copies of documents passing through it; subsequent requests may
be satisfied from the cache if certain conditions are met[.](https://en.wikipedia.org/wiki/Web_cache)


### Why?

Simple, you want faster web applications,
use of the httpcache gives you more than 20 times performance advantage than other websites.

So, should I use cache for everyhing?
 * **NO**, use cache for a handler when you think that a specific page will not change soon,
 httpcache refreshes data on a custom time pass limit*

 Ideas for cache: blogs, index pages, contact pages, about me websites...


`httpcache` gives you the ability to cache certain handlers or the whole website,
it's up to you where you want to apply cache!

Quick Start
-----------

The only requirement is the [Go Programming Language](https://golang.org/dl).

```bash
$ go get -u github.com/geekypanda/httpcache/...
```

### What's inside?

- `Cache` & `CacheFasthttp` functions, convert any type of Handler to `cached Handler`.

**For distributed applications only:**
- `ListenAndServe` function, starts the remote cache service on a specific network address.
- `CacheRemote` & `CacheRemoteFasthttp` functions, convert any type of Handler
which hosted in the client-side machine, to a `cached Handler`
 which communicates with the remote cache server's Handler.


### Mime support?

In short terms, **any data with any [content type](http://www.freeformatter.com/mime-types-list.html) is cached**.

Some of them are...

- `application/json`
- `text/html`
- `text/plain`
- `text/xml`
- `text/javascript (JSONP)``
- `application/octet-stream`
- `application/pdf`
- `image/jpeg`
- `image/png`
- `image/gif`
- `image/bmp`
- `image/svg+xml`
- `image/x-icon`


### Usage

```go
package main

import (
	"net/http"
	"time"

	"github.com/geekypanda/httpcache"
)

func main() {
	// The only thing that separates your handler to be cached is just
    // ONE function wrapper
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


```

> Tip: To cache the whole website for a certain time, pass the `Cache` at the `mux`.
`http.ListenAndServe(":8080", http.Cache(http.DefaultServeMux, 5 *time.Minute))`

### Next?

- Navigate to the [_examples](https://github.com/GeekyPanda/httpcache/tree/master/_examples) folder to see a small overview of cached templates.

- I didn't see any usage example of fasthttp here, why?
 * I want to keep the README as small as possible, but you can find an example of fasthttp-usage in the [_examples/fasthttp](https://github.com/GeekyPanda/httpcache/tree/master/_examples/fasthttp) folder.


- What if I split my application in several servers, does this library has support for distributed apps?
 * Yes, again with one-function, go to [httpcache_test.go:TestCacheDistributed](https://github.com/GeekyPanda/httpcache/blob/master/httpcache_test.go#L80) and see how.



People
------------

The authors of httpcache project are:

- [@GeekyPanda](https://github.com/GeekyPanda) has over a decade's experience on network and distributed systems.
- [@kataras](https://github.com/kataras) is the author of the fastest GoLang web framework.


License
------------

Unless otherwise noted, the httpcache source files are distributed under the
terms of the MIT License.

License can be found [here](LICENSE).

[Travis Widget]: https://img.shields.io/travis/GeekyPanda/httpcache.svg?style=flat-square
[Travis]: http://travis-ci.org/GeekyPanda/httpcache
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/GeekyPanda/httpcache/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/version-0.0.2-blue.svg?style=flat-square
[Release]: https://github.com/GeekyPanda/httpcache/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]:  https://gitter.im/go-httpcache/Lobby
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/GeekyPanda/httpcache
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-gray.svg?style=flat-square
