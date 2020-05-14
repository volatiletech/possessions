package possessions

import (
	"bufio"
	"context"
	"net"
	"net/http"

	"github.com/pkg/errors"
)

// CTXKeyPossessions key for caching sessions
type CTXKeyPossessions struct{}

// UnderlyingResponseWriter allows wrapping responsewriters to be able to
// hand the possessions.responseWriter back to possessions for session
// management while still being able to use their own custom one.
type UnderlyingResponseWriter interface {
	UnderlyingResponseWriter() http.ResponseWriter
}

type possesionsWriter struct {
	// the wrapped responsewriter
	underlying http.ResponseWriter
	// overseer is responsible for writing sessions in some way to the client
	overseer Overseer

	// State for the request
	// context is from the request, we only keep this for the duration
	// of the request for cancellation/deadlines for things that occur
	// when writing out sessions so it should be okay
	ctx        context.Context
	session    Session
	hasWritten bool
	events     []Event
}

func newResponseWriter(ctx context.Context, w http.ResponseWriter, overseer Overseer) *possesionsWriter {
	return &possesionsWriter{
		underlying: w,
		overseer:   overseer,
		ctx:        ctx,
	}
}

// Header retrieves the underlying headers
func (r *possesionsWriter) Header() http.Header {
	return r.underlying.Header()
}

// Hijack implements the http.Hijacker interface by calling the
// underlying implementation if available.
func (r *possesionsWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.underlying.(http.Hijacker)
	if ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("possessions: underlying ResponseWriter does not support hijacking")
}

// WriteHeader writes the header, but in order to handle errors from the
// underlying ClientStateReadWriter, it has to panic.
func (r *possesionsWriter) WriteHeader(code int) {
	if !r.hasWritten {
		if err := r.writeClientState(r.ctx); err != nil {
			panic(err)
		}
	}
	r.underlying.WriteHeader(code)
}

// Write ensures that the client state is written before any writes
// to the body occur (before header flush to http client)
func (r *possesionsWriter) Write(b []byte) (int, error) {
	if !r.hasWritten {
		if err := r.writeClientState(r.ctx); err != nil {
			return 0, err
		}
	}
	return r.Write(b)
}

// UnderlyingResponseWriter for this instance
func (r *possesionsWriter) UnderlyingResponseWriter() http.ResponseWriter {
	return r.underlying
}

func (r *possesionsWriter) writeClientState(ctx context.Context) error {
	if err := r.overseer.WriteState(r.underlying, r.session, r.events); err != nil {
		return err
	}
	r.hasWritten = true

	return nil
}

// OverseeingMiddleware enables the use of sessions in this package by allowing
// read and writes of client state during the request.
type OverseeingMiddleware struct {
	overseer Overseer
}

type oversight struct {
	handler  http.Handler
	overseer Overseer
}

// NewOverseeingMiddleware constructs a middleware
func NewOverseeingMiddleware(overseer Overseer) OverseeingMiddleware {
	return OverseeingMiddleware{
		overseer: overseer,
	}
}

// Wrap a handler
func (o OverseeingMiddleware) Wrap(h http.Handler) http.Handler {
	return oversight{
		handler:  h,
		overseer: o.overseer,
	}
}

// ServeHTTP implements http.Handler
func (o oversight) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, err := o.overseer.ReadState(r)
	if err != nil {
		panic(errors.Wrap(err, "failed to read session state"))
	}

	ctx := context.WithValue(r.Context(), CTXKeyPossessions{}, session)
	w = newResponseWriter(ctx, w, o.overseer)
	r = r.WithContext(ctx)
	o.handler.ServeHTTP(w, r)
}
