package fhttp

import (
	"github.com/geekypanda/httpcache/internal"
	"github.com/valyala/fasthttp"
)

// Validators are introduced to implement the RFC about cache (https://tools.ietf.org/html/rfc7234#section-1.1).

// PreValidator like middleware, executes before the cache action begins, if a callback returns false
// then this specific cache action, with specific request, is ignored and the real (original)
// handler is executed instead.
//
// I'll not add all specifications here I'll give the oportunity (public API in the httpcache package-level)
// to the end-user to specify her/his ignore rules too (ignore-only for now).
//
// Each package, nethttp and fhttp should implement their own encapsulations because of different request object.
//
// One function, accepts the request and returns false if should be denied/ignore, otherwise true.
// if at least one return false then the original handler will execute as it's
// and the whole cache action(set & get) should be ignored, it will be never go to the step of post-cache validations.
type PreValidator func(*fasthttp.RequestCtx) bool

// PostValidator type is is introduced to implement the second part of the RFC about cache.
//
// same signature with PreValidator in order to align with the net/htttp package
//
// Q: What's the difference between this and a PreValidator?
// A: PreValidator runs BEFORE trying to get the cache, it cares only for the request
//    and if at least one PreValidator returns false then it just runs the original handler and stop there, at the other hand
//    a PostValidator runs if all PreValidators returns true and original handler is executed but with the original handler's response.
//
// If a function of type of PostValidator returns true then the (shared-always) cache is allowed to be stored.
type PostValidator func(*fasthttp.RequestCtx) bool

// DefaultPreValidators are like middleware, execute before the cache action begins, if a callback returns false
// then this specific cache action, with specific request, is ignored and the real (original)
// handler is executed instead.
//
// I'll not add all specifications here I'll give the oportunity (public API in the httpcache package-level)
// to the end-user to specify her/his ignore rules too (ignore-only for now).
//
// Each package, nethttp and fhttp should implement their own encapsulations because of different request object.
//
// One function, accepts the request and returns false if should be denied/ignore, otherwise true.
// if at least one return false then the original handler will execute as it's
// and the whole cache action(set & get) should be ignored, it will be never go to the step of post-cache validations.
//
// In short words, if at least one of them return false then deny the cache at all,
// don't bother to get or set anything, execute the original handler.
//
// DefaultPreValidators is a list of the default pre-cache validators which exists in ALL handlers, local and remote.
var DefaultPreValidators = []PreValidator{
	// #1 A shared cache MUST NOT use a cached response to a request with an
	// Authorization header field
	func(reqCtx *fasthttp.RequestCtx) bool {
		get := func(key string) string {
			return string(reqCtx.Request.Header.Peek(key))
		}
		return get("Authorization") == "" &&
			get("Proxy-Authenticate") == ""
	},
	// #2 "must-revalidate" and/or
	// "s-maxage" response directives are not allowed to be served stale
	// (Section 4.2.4) by shared caches.  In particular, a response with
	// either "max-age=0, must-revalidate" or "s-maxage=0" cannot be used to
	// satisfy a subsequent request without revalidating it on the origin
	// server.
	func(reqCtx *fasthttp.RequestCtx) bool {
		get := func(key string) string {
			return string(reqCtx.Request.Header.Peek(key))
		}
		return get("Must-Revalidate") == "" &&
			get("S-Maxage") != "0" &&
			get("Max-Age") != "0" &&
			get(internal.NoCacheHeader) != "true"
	},
}

// DefaultPostValidators variable is introduced to implement the second part of the RFC about cache.
//
// Q: What's the difference between this and a PreValidators?
// A: PreValidators runs BEFORE trying to get the cache, it cares only for the request
//    and if at least one PreValidator returns false then it just runs the original handler and stop there, at the other hand
//    a PostValidator runs if all PreValidators returns true and original handler is executed but with a response recorder,
//    also the PostValidators should ALL return true to store the cached response.
//    Both types are needed for special cases, it's no PreValidators vs PostValidators,
//    they are different things because they are executing on different order and cases.
//    Last, the PostValidator accepts a `ResponseRecorder` to be able to catch the original handler's response,
//    the PreValidator checks only for request.
//
//
// DefaultPostValidators is a list of the default post-cache validators which exists in ALL handlers, local and remote.
var DefaultPostValidators = []PostValidator{}

// NoCache called when a particular handler is not valid for cache.
// If this function called inside a handler then the handler is not cached
// even if it's surrounded with the Cache/CacheFunc wrappers.
func NoCache(reqCtx *fasthttp.RequestCtx) {
	reqCtx.Response.Header.Set(internal.NoCacheHeader, "true")
}

// Validator contains the validators used on the consumer handler
type Validator struct {
	// preValidators a list of PreValidator functions, execute before real cache begins
	// if at least one of them returns false then the original handler will execute as it's
	// and the whole cache action(set & get) will be skipped for this specific client's request.
	//
	// Read-only 'runtime'
	preValidators []PreValidator

	// postValidators a list of PostValidator functions, execute after the original handler is executed with the response recorder
	// and exactly before this cached response is saved,
	// if at least one of them returns false then the response will be not saved for this specific client's request.
	//
	// Read-only 'runtime'
	postValidators []PostValidator
}

// DefaultValidator returns a new validator which contains the default pre and post cache validators
func DefaultValidator() *Validator {
	return &Validator{
		preValidators:  DefaultPreValidators,
		postValidators: DefaultPostValidators,
	}
}

// EmptyValidator returns a new empty validator container
func EmptyValidator() *Validator {
	return &Validator{}
}

// ClainWhen adds a pre validator and returns itsself
func (v *Validator) ClainWhen(preValidatorsFn ...PreValidator) *Validator {
	v.preValidators = append(v.preValidators, preValidatorsFn...)
	return v
}

// ValidWhen adds a post validator and returns itsself
func (v *Validator) ValidWhen(postValidatorsFn ...PostValidator) *Validator {
	v.postValidators = append(v.postValidators, postValidatorsFn...)
	return v
}

// claim returns true if incoming request can claim for a cached handler
// the original handler should run as it is and exit
func (v *Validator) claim(reqCtx *fasthttp.RequestCtx) bool {
	// check for pre-cache validators, if at least one of them return false
	// for this specific request, then skip the whole cache
	for _, shouldProcess := range v.preValidators {
		if !shouldProcess(reqCtx) {
			return false
		}
	}
	return true
}

// valid returns true if incoming request and post-response from the original handler
// is valid to be store to the cache, if not(false) then the consumer should just exit
// otherwise(true) the consumer should store the cached response
func (v *Validator) valid(reqCtx *fasthttp.RequestCtx) bool {
	// check if it's a valid response, if it's not then just return.
	for _, valid := range v.postValidators {
		if !valid(reqCtx) {
			return false
		}
	}
	return true
}
