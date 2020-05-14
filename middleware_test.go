package possessions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRefreshMiddleware(t *testing.T) {
	t.Parallel()

	w := newResponseWriter(context.Background(), httptest.NewRecorder(), nil)
	r := httptest.NewRequest("GET", "/", nil)

	called := false
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	refresh := NewRefreshMiddleware()
	refresh.Wrap(httpHandler).ServeHTTP(w, r)

	if len(w.events) != 1 {
		t.Error("expected one event, got:", len(w.events))
	}
	if k := w.events[0].Kind; k != EventRefresh {
		t.Error("the event in the responsewriter should be a refresh")
	}
	if !called {
		t.Error("the handler should have been called")
	}
}
