package fhttp

import (
	"time"

	"github.com/geekypanda/httpcache/internal"
	"github.com/valyala/fasthttp"
)

// Handler the fasthttp cache service handler
type Handler struct {
	// Validator optional validators for pre cache and post cache actions
	//
	// See more at validator.go
	Validator *Validator

	// bodyHandler the original route's handler
	bodyHandler fasthttp.RequestHandler

	// entry is the memory cache entry
	entry *internal.Entry
}

// NewHandler returns a new cached handler
func NewHandler(bodyHandler fasthttp.RequestHandler,
	expireDuration time.Duration) *Handler {
	e := internal.NewEntry(expireDuration)

	return &Handler{
		Validator:   DefaultValidator(),
		bodyHandler: bodyHandler,
		entry:       e,
	}
}

func (h *Handler) ServeHTTP(reqCtx *fasthttp.RequestCtx) {

	// check for pre-cache validators, if at least one of them return false
	// for this specific request, then skip the whole cache
	if !h.Validator.claim(reqCtx) {
		h.bodyHandler(reqCtx)
		return
	}

	// check if we have a stored response( it is not expired)
	res, exists := h.entry.Response()
	if !exists {
		// if it's not valid then execute the original handler
		h.bodyHandler(reqCtx)

		// check if it's a valid response, if it's not then just return.
		if !h.Validator.valid(reqCtx) {
			return
		}

		// no need to copy the body, its already done inside
		body := reqCtx.Response.Body()
		if len(body) == 0 {
			// if no body then just exit
			return
		}

		// and re-new the entry's response with the new data
		statusCode := reqCtx.Response.StatusCode()
		contentType := string(reqCtx.Response.Header.ContentType())

		// check for an expiration time if the
		// given expiration was not valid &
		// update the response & release the recorder
		h.entry.Reset(statusCode, contentType, body, GetMaxAge(reqCtx))
		return
	}

	// if it's valid then just write the cached results
	reqCtx.SetStatusCode(res.StatusCode())
	reqCtx.SetContentType(res.ContentType())
	reqCtx.SetBody(res.Body())
}
