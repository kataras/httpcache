// The MIT License (MIT)
//
// Copyright (c) 2016 GeekyPanda
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package httpcache

import (
	"net/http"
	"sync"
)

var rpool = sync.Pool{}

func acquireResponseRecorder(underline http.ResponseWriter) *responseRecorder {
	v := rpool.Get()
	var res *responseRecorder
	if v != nil {
		res = v.(*responseRecorder)
	} else {
		res = &responseRecorder{}
	}
	res.underline = underline
	return res
}

func releaseResponseRecorder(res *responseRecorder) {
	res.underline = nil
	res.statusCode = 0
	res.chunks = res.chunks[0:0]
	rpool.Put(res)
}

type responseRecorder struct {
	underline  http.ResponseWriter
	chunks     [][]byte // 2d because .Write can be called more than one time in the same handler and we want to cache all of them
	statusCode int      // the saved status code which will be used from the cache service
}

// getBody joins the chunks to one []byte slice, this is the full body
func (res *responseRecorder) getBody() []byte {
	var body []byte
	for i := range res.chunks {
		body = append(body, res.chunks[i]...)
	}
	return body
}

// Header returns the header map that will be sent by
// WriteHeader. Changing the header after a call to
// WriteHeader (or Write) has no effect unless the modified
// headers were declared as trailers by setting the
// "Trailer" header before the call to WriteHeader (see example).
// To suppress implicit response headers, set their value to nil.
func (res *responseRecorder) Header() http.Header {
	return res.underline.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
//
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
func (res *responseRecorder) Write(contents []byte) (int, error) {
	if res.statusCode == 0 { // if not setted set it here
		res.WriteHeader(http.StatusOK)
	}
	res.chunks = append(res.chunks, contents)
	return res.underline.Write(contents)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (res *responseRecorder) WriteHeader(statusCode int) {
	if res.statusCode == 0 { // set it only if not setted already, we don't want logs about multiple sends
		res.statusCode = statusCode
		res.underline.WriteHeader(statusCode)
	}

}
