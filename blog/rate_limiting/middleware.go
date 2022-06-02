package rate_limiting

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/render"
	"go.uber.org/zap"
	"hxann.com/blog/blog/resp"
)

var (
	_            http.Handler = &httpRateLimiterHandler{}
	_            Extractor    = &httpHeaderExtractor{}
	stateStrings              = map[State]string{
		Allow: "Allow",
		Deny:  "Deny",
	}
)

const (
	rateLimitingTotalRequests = "Rate-Limiting-Total-Requests"
	rateLimitingState         = "Rate-Limiting-State"
	rateLimitingExpiresAt     = "Rate-Limiting-Expires-At"
)

// Extractor represents the way we will extract a key from an HTTP request, this
// could be a value from a header, request path, method used, user
// authentication information, any information that is available at the HTTP
// request that wouldn't cause side effects if it was collected (this object
// shouldn't read the body of the request).
type Extractor interface {
	Extract(r *http.Request) (string, error)
}

type httpHeaderExtractor struct {
	headers []string
}

// Extract extracts a collection of http headers and joins them to build the key
// that will be used for rate limiting. You should use headers that are
// guaranteed to be unique for a client.
func (h *httpHeaderExtractor) Extract(r *http.Request) (string, error) {
	values := make([]string, 0, len(h.headers))

	for _, key := range h.headers {
		// if we can't find a value for the headers, give up and return an error.
		if value := strings.TrimSpace(r.Header.Get(key)); value == "" {
			return "", fmt.Errorf("the header %v must have a value set", key)
		} else {
			values = append(values, value)
		}
	}

	return strings.Join(values, "-"), nil
}

// NewHTTPHeadersExtractor creates a new HTTP header extractor
func NewHTTPHeadersExtractor(headers ...string) Extractor {
	return &httpHeaderExtractor{headers: headers}
}

// RateLimiterConfig holds the basic config we need to create a middleware
// http.Handler object that performs rate limiting before offloading the request
// to an actual handler.
type RateLimiterConfig struct {
	Extractor   Extractor
	Strategy    Strategy
	Expiration  time.Duration
	MaxRequests uint64
}

// NewHTTPRateLimiterHandler wraps an existing http.Handler object performing
// rate limiting before sending the request to the wrapped handler. If any
// errors happen while trying to rate limit a request or if the request is
// denied, the rate limiting handler will send a response to the client and will
// not call the wrapped handler.
func NewHTTPRateLimiterHandler(originalHandler http.Handler, sugar *zap.SugaredLogger, config *RateLimiterConfig) http.Handler {
	return &httpRateLimiterHandler{
		handler: originalHandler,
		sugar:   sugar,
		config:  config,
	}
}

type httpRateLimiterHandler struct {
	handler http.Handler
	sugar   *zap.SugaredLogger
	config  *RateLimiterConfig
}

// ServeHTTP performs rate limiting with the configuration it was provided and
// if there were not errors and the request was allowed it is sent to the
// wrapped handler. It also adds rate limiting headers that will be sent to the
// client to make it aware of what state it is in terms of rate limiting.
func (h *httpRateLimiterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key, err := h.config.Extractor.Extract(r)
	if err != nil {
		h.sugar.Debugf("failed to collect rate limiting key from request: %v", err)
		render.Render(w, r, resp.ErrInvalidRequest(fmt.Errorf("failed to collect rate limiting key from request: %w", err)))
		return
	}

	result, err := h.config.Strategy.Run(r.Context(), &Request{
		Key:      key,
		Limit:    h.config.MaxRequests,
		Duration: h.config.Expiration,
	})

	if err != nil {
		h.sugar.Errorf("failed to run rate limiting for request: %v", err)
		render.Render(w, r, resp.ErrInternal(fmt.Errorf("failed to run rate limiting for request: %w", err)))
		return
	}

	// set the rate limiting headers both on allow or deny results so the client
	// knows what is going on
	w.Header().Set(rateLimitingTotalRequests, strconv.FormatUint(result.TotalRequests, 10))
	w.Header().Set(rateLimitingState, stateStrings[result.State])
	w.Header().Set(rateLimitingExpiresAt, result.ExpiresAt.Format(time.RFC3339))

	// when the state is Deny, just return a 429 response to the client and stop
	// the request handling flow
	if result.State == Deny {
		render.Render(w, r, resp.ErrTooManyRequest(errors.New("you have sent too many requests to this service, slow down please")))
		return
	}

	// if the request was not denied we assume it was allowed and call the wrapped
	// handler. by leaving this to the end we make sure the wrapped handler is
	// only called once and doesn't have to worry about any rate limiting at all
	// (it doesn't even have to know there was rate limiting happening for this
	// request) as we have already set the headers, so when the handler flushes
	// the response the headers above will be sent.
	h.handler.ServeHTTP(w, r)
}
