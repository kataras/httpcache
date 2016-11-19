package rule

import (
	"github.com/geekypanda/httpcache/internal"
	"github.com/valyala/fasthttp"
)

// The HeaderPredicate should be alived on each of $package/rule BUT GOLANG DOESN'T SUPPORT type alias and I don't want to have so many copies arround
// read more at ../../ruleset.go

// headerRule is a Rule witch receives and checks for a header predicates
// request headers on Claim and response headers on Valid.
type headerRule struct {
	claim internal.HeaderPredicate
	valid internal.HeaderPredicate
}

var _ Rule = &headerRule{}

// Header returns a new rule witch claims and execute the post validations trough headers
func Header(claim internal.HeaderPredicate, valid internal.HeaderPredicate) Rule {
	if claim == nil {
		claim = internal.EmptyHeaderPredicate
	}

	if valid == nil {
		valid = internal.EmptyHeaderPredicate
	}

	return &headerRule{
		claim: claim,
		valid: valid,
	}
}

// HeaderClaim returns a header rule which cares only about claiming (pre-validation)
func HeaderClaim(claim internal.HeaderPredicate) Rule {
	return Header(claim, nil)
}

// HeaderValid returns a header rule which cares only about valid (post-validation)
func HeaderValid(valid internal.HeaderPredicate) Rule {
	return Header(nil, valid)
}

// Claim validator
func (h *headerRule) Claim(reqCtx *fasthttp.RequestCtx) bool {
	j := func(reqCtx *fasthttp.RequestCtx) bool {
		get := func(key string) string {
			return string(reqCtx.Request.Header.Peek(key))
		}

		return h.claim(get)
	}
	return j(reqCtx)
}

// Valid validator
func (h *headerRule) Valid(reqCtx *fasthttp.RequestCtx) bool {
	j := func(reqCtx *fasthttp.RequestCtx) bool {
		get := func(key string) string {
			return string(reqCtx.Response.Header.Peek(key))
		}

		return h.valid(get)
	}
	return j(reqCtx)
}
