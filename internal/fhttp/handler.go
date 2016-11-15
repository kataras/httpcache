package fhttp

import (
	"time"

	"github.com/geekypanda/httpcache/internal"
	"github.com/valyala/fasthttp"
)

// Handler the fasthttp cache service handler
type Handler struct {
	// Entry is the cache entry
	Entry *internal.Entry
	// bodyHandler the original route's handler
	bodyHandler fasthttp.RequestHandler
}

// NewHandler returns a new cached handler
func NewHandler(bodyHandler fasthttp.RequestHandler,
	expireDuration time.Duration) *Handler {
	e := internal.NewEntry(expireDuration)

	return &Handler{
		Entry:       e,
		bodyHandler: bodyHandler,
	}
}

func (h *Handler) ServeHTTP(reqCtx *fasthttp.RequestCtx) {
	// check if is valid
	res, valid := h.Entry.Response()
	if !valid {
		// if it's not valid then execute the original handler
		h.bodyHandler(reqCtx)
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
		h.Entry.Reset(statusCode, contentType, body, GetMaxAge(reqCtx))
		return
	}

	// if it's valid then just write the cached results
	reqCtx.SetStatusCode(res.StatusCode())
	reqCtx.SetContentType(res.ContentType())
	reqCtx.SetBody(res.Body())
}
