package fhttp

import (
	"github.com/geekypanda/httpcache/cfg"
	"github.com/geekypanda/httpcache/fhttp/rule"
	"github.com/geekypanda/httpcache/ruleset"
	"github.com/valyala/fasthttp"
)

// here we are f... copying source code because golang doesn't
// supports alias (for HeaderPredicate for example,
// I must move it outside of the header rule to the internal package which makes no sense but......)
//  as we wanted to do and we go back the alias feature.............

// DefaultRuleSet is a list of the default pre-cache validators
// which exists in ALL handlers, local and remote.
var DefaultRuleSet = rule.Chained(
	// #1 A shared cache MUST NOT use a cached response to a request with an
	// Authorization header field
	rule.HeaderClaim(ruleset.AuthorizationRule),
	// #2 "must-revalidate" and/or
	// "s-maxage" response directives are not allowed to be served stale
	// (Section 4.2.4) by shared caches.  In particular, a response with
	// either "max-age=0, must-revalidate" or "s-maxage=0" cannot be used to
	// satisfy a subsequent request without revalidating it on the origin
	// server.
	rule.HeaderClaim(ruleset.MustRevalidateRule),
	rule.HeaderClaim(ruleset.ZeroMaxAgeRule),
	// #3 custom No-Cache header used inside this library
	// for BOTH request and response (after get-cache action)
	rule.Header(ruleset.NoCacheRule, ruleset.NoCacheRule),
)

// NoCache called when a particular handler is not valid for cache.
// If this function called inside a handler then the handler is not cached.
func NoCache(reqCtx *fasthttp.RequestCtx) {
	reqCtx.Response.Header.Set(cfg.NoCacheHeader, "true")
}
