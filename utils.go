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
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	failStatus         = 400
	successStatus      = 200
	contentText        = "text/plain; charset=utf-8"
	contentTypeHeader  = "Content-Type"
	statusCodeHeader   = "Status"
	cacheControlHeader = "Cache-Control"
	//remote
	queryCacheKey         = "cache_key"
	queryCacheDuration    = "cache_duration"
	queryCacheStatusCode  = "cache_status_code"
	queryCacheContentType = "cache_content_type"
	requestCacheTimeout   = 5 * time.Second
	methodGet             = "GET"
	methodPost            = "POST"
	methodDelete          = "DELETE"
)

var (
	methodGetBytes    = []byte(methodGet)
	methodPostBytes   = []byte(methodPost)
	methodDeleteBytes = []byte(methodDelete)
)

var minimumAllowedCacheDuration = 2 * time.Second

func validateCacheDuration(expiration time.Duration) time.Duration {
	if expiration <= minimumAllowedCacheDuration {
		expiration = minimumAllowedCacheDuration * 2
	}
	return expiration
}

func validateStatusCode(statusCode int) int {
	if statusCode <= 0 {
		statusCode = successStatus
	}
	return statusCode
}

func validateContentType(cType string) string {
	if cType == "" {
		cType = contentText
	}
	return cType
}

var maxAgeExp = regexp.MustCompile(`maxage=(\d+)`)

// parseMaxAge parses the max age from the receiver paramter, "cache-control" header
// returns seconds as int64
// if header not found or parse failed then it returns -1
func parseMaxAge(header string) int64 {
	if header == "" {
		return -1
	}
	m := maxAgeExp.FindStringSubmatch(header)
	if len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			return int64(v)
		}
	}
	return -1
}

// remote

type remoteURI struct {
	remoteURL,
	key,
	method,
	scheme,
	host,
	cacheDurationStr,
	statusCodeStr,
	contentType string
}

// String returns the full url which should be passed to get a cache entry response back
// (it could be setted by server too but we need some client-freedom on the requested key)
// in order to be sure that the registered cache entries are unique among different clients with the same key
// note1: we do it manually*,
// note2: on fasthttp that is not required because the query args added as expected but we will use it there too to be align with net/http
func (r remoteURI) String() string {

	remoteURL := r.remoteURL

	// fasthttp appends the "/" in the last uri (with query args also, that's probably a fasthttp bug which I'll fix later)
	// for now lets make that check:

	if !strings.HasSuffix(remoteURL, "/") {
		remoteURL += "/"
	}

	// validate the remoteURL, should contains a scheme, if not then check if the client has given a scheme and if so
	// use that for the server too
	if !strings.Contains(remoteURL, "://") {

		if strings.Contains(remoteURL, ":443") || strings.Contains(remoteURL, ":https") {
			remoteURL = "https://" + remoteURL
		} else if r.scheme != "" {
			//check for client's scheme and set it
			remoteURL = r.scheme + "://" + remoteURL
		}

	}

	s := remoteURL + "?" +
		queryCacheKey + "=" + url.QueryEscape(r.method+r.scheme+r.host+r.key)
	if r.cacheDurationStr != "" {
		s += "&" + queryCacheDuration + "=" + url.QueryEscape(r.cacheDurationStr)
	}
	if r.statusCodeStr != "" {
		s += "&" + queryCacheStatusCode + "=" + url.QueryEscape(r.statusCodeStr)
	}
	if r.contentType != "" {
		s += "&" + queryCacheContentType + "=" + url.QueryEscape(r.contentType)
	}
	return s
}
