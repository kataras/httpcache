package nethttp

import (
	"net/http"
)

// Denier is introduced to implement the RFC about cache (https://tools.ietf.org/html/rfc7234#section-1.1)
// Like (begin-only) middleware, execute before the cache action begins, if the callback returns true
// then this specific cache action, with specific request, is ignored and the real (original)
// handler is executed instead.
//
// I'll not add all specifications here I'll give the oportunity (public API in the httpcache package-level)
// to the end-user to specify her/his ignore rules too (ignore-only for now).
//
// Each package, nethttp and fhttp should implement their own encapsulations because of different request object.
//
// One function, accepts the request and returns true if should be denied/ignore, otherwise false.
// if return true then the original handler will execute as it's
// and the whole cache action(set & get) should be ignored.
// =================
// Note: I am not name that as 'Skip' because I think that we need MIDL functions which will come to play one step before
// the cache response will be set after these tests are passed
// so for example if we have a 401 response from our original handler, then we should not add this body to the cache,
// so maybe these type of handlers which will accept the response recorder (or fasthttp.RequestCtx) will be named as 'Skipers'.
// =================
//
// In short words, if return true deny the cache at all, don't bother to get or set anything, execute the original handler.
type Denier func(*http.Request) bool

// DefaultDeniers is a list of the default deniers which exists in ALL handlers, local and remote.
/// TODO: Change the implementation and introduce other pattern to make these come
// as one part instead of two because the bodyHandler(original handler) and the Deniers
// are ALWAYS come together and the execution of the 'cache handler action' depends on these deniers
// ---- I made them fast in order to not forget the idea of the Deniers I just though,
//  it will be working but these may change on the future,
//  so for now just let the package
// do its own things and don't try to change or add new Deniers (YET)
// and for one more reason, that we have many dublications between handler.go and client.go,
// once again the bodyHandler MUST come with the Deniers BUT
// the execution of both of them are lived inside the handler ACTION.ServeHTTP -----
//
// DefaultDeniers can be changed BEFORE servers run, so no constant, at LEAST FOR NOW
var DefaultDeniers = []Denier{
	// #1 A shared cache MUST NOT use a cached response to a request with an
	// Authorization header field
	func(r *http.Request) bool {
		h := r.Header
		return h.Get("Authorization") != "" ||
			h.Get("Proxy-Authenticate") != ""
	},
	// #2 "must-revalidate" and/or
	// "s-maxage" response directives are not allowed to be served stale
	// (Section 4.2.4) by shared caches.  In particular, a response with
	// either "max-age=0, must-revalidate" or "s-maxage=0" cannot be used to
	// satisfy a subsequent request without revalidating it on the origin
	// server.
	func(r *http.Request) bool {
		h := r.Header
		return h.Get("must-revalidate") != "" ||
			h.Get("s-maxage") == "0" ||
			h.Get("max-age") == "0" ||
			h.Get("X-No-Cache") != "" // this is a custom header, for any case ( I use that in my projects so I recommend and introduce this here too*)
	},
}

// NOTE: We will need to change that or intoduce another set of Deniers to be executed in the MIDLE (before set the CACHED RESPONSE)
// in order to check the RESPONSE STATUS CODE ( for example if it was 401, unauthenticated then we should not cache this response)
