package nethttp

import (
	"net/http"
	"time"

	"github.com/geekypanda/httpcache/internal"
)

// Handler the local cache service handler
type Handler struct {
	// Entry is the cache entry
	Entry *internal.Entry
	// bodyHandler the original route's handler
	bodyHandler http.Handler
}

// NewHandler returns a new cached handler
func NewHandler(bodyHandler http.Handler,
	expireDuration time.Duration) *Handler {

	e := internal.NewEntry(expireDuration)
	return &Handler{
		Entry:       e,
		bodyHandler: bodyHandler,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check if is valid
	res, valid := h.Entry.Response()
	if !valid {
		// if it's not valid then execute the original handler
		// with our custom response recorder response writer
		// because the net/http doesn't give us
		// a built'n way to get the status code & body
		recorder := AcquireResponseRecorder(w)
		h.bodyHandler.ServeHTTP(recorder, r)
		// no need to copy the body, its already done inside
		body := recorder.Body()
		if len(body) == 0 {
			// if no body then just exit
			return
		}

		// check for an expiration time if the
		// given expiration was not valid &
		// update the response & release the recorder
		h.Entry.Reset(recorder.StatusCode(), recorder.ContentType(), body, GetMaxAge(r))
		ReleaseResponseRecorder(recorder)
		return
	}

	// if it's valid then just write the cached results
	w.Header().Set("Content-Type", res.ContentType())
	w.WriteHeader(res.StatusCode())
	w.Write(res.Body())
}
