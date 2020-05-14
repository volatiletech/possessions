package possessions

import "net/http"

// RefreshMiddleware refreshes sessions on each request
type RefreshMiddleware struct{}
type refreshSession struct {
	handler http.Handler
}

// NewRefreshMiddleware creates a refresh middleware
func NewRefreshMiddleware() RefreshMiddleware {
	return RefreshMiddleware{}
}

// Wrap wraps a handler with refreshing middleware
func (r RefreshMiddleware) Wrap(h http.Handler) http.Handler {
	return refreshSession{handler: h}
}

func (r refreshSession) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	pw := getResponseWriter(w)
	pw.events = append(pw.events, Event{
		Kind: EventRefresh,
	})

	r.handler.ServeHTTP(w, req)
}
