package possessions

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

var (
	rgxSetCookie = regexp.MustCompile(`id=[A-Za-z0-9\-]+; HttpOnly; Secure`)
	// Do assign to nothing to check if implementation of StorageOverseer is complete
	_ Overseer = StorageOverseer{}
)

func TestStorageOverseerNew(t *testing.T) {
	t.Parallel()

	opts := CookieOptions{
		MaxAge:   2,
		Secure:   true,
		HTTPOnly: true,
		Name:     "id",
	}

	mem, err := NewDefaultMemoryStorer()
	if err != nil {
		t.Fatal(err)
	}

	s := NewStorageOverseer(opts, mem)
	if err != nil {
		t.Error(err)
	}

	if s.options.MaxAge != 2 {
		t.Error("expected client expiry to be 2")
	}

	if s.options.Secure != true {
		t.Error("expected secure to be true")
	}

	if s.options.HTTPOnly != true {
		t.Error("expected httpOnly to be true")
	}
}

func TestReadState(t *testing.T) {
	t.Parallel()

	uuid := "816a1acb-73aa-4a75-bbeb-f371bdad40e8"
	r := httptest.NewRequest("GET", "http://localhost", nil)

	m, err := NewDefaultMemoryStorer()
	if err != nil {
		t.Error(err)
	}
	s := NewStorageOverseer(NewCookieOptions(), m)
	err = m.Set(r.Context(), uuid, `{"key":"value"}`)
	if err != nil {
		t.Error(err)
	}

	cookie := new(http.Cookie)
	cookie.Name = "id"
	cookie.Value = uuid

	r.AddCookie(cookie)

	sess, err := s.ReadState(r)
	if err != nil {
		t.Error("cannot read state:", err)
	}

	val, hasKey := sess.Get("key")
	if !hasKey {
		t.Error("couldnt find test key")
	}
	if val != "value" {
		t.Error("value doesnt equal value, got:", val)
	}
}

func TestWriteState(t *testing.T) {
	t.Parallel()

	uuid := "816a1acb-73aa-4a75-bbeb-f371bdad40e8"
	r := httptest.NewRequest("GET", "http://localhost", nil)
	ctx := r.Context()

	m, err := NewDefaultMemoryStorer()
	if err != nil {
		t.Error(err)
	}
	s := NewStorageOverseer(NewCookieOptions(), m)
	rec := httptest.NewRecorder()
	w := newResponseWriter(r.Context(), rec, s)

	ev := Event{Kind: EventSet, Key: "key", Val: "value"}

	sess := session{ID: uuid, Values: map[string]string{}}
	s.WriteState(ctx, w, sess, []Event{ev})
	val, err := m.Get(r.Context(), uuid)
	if err != nil {
		t.Error(err)
	}

	if val != `{"key":"value"}` {
		t.Error("invalid value in memory storer session")
	}

	w.WriteHeader(http.StatusOK)
	if rec.Header().Get("Set-Cookie") == "" {
		t.Error("cookie value not set")
	}
}

func TestApplyEvents(t *testing.T) {
	t.Parallel()

	uuid := "816a1acb-73aa-4a75-bbeb-f371bdad40e8"
	sess := session{ID: uuid, Values: map[string]string{}}

	events := []Event{
		{Kind: EventSet, Key: "key1", Val: "value1"},
		{Kind: EventSet, Key: "key2", Val: "value2"},
		{Kind: EventSet, Key: "key3", Val: "value3"},
		{Kind: EventSet, Key: "key4", Val: "value4"},
		{Kind: EventDel, Key: "key2"},
		{Kind: EventDelAll, Keys: []string{"key1", "key3"}},
		{Kind: EventRefresh},
	}

	doRefresh := applyEvents(sess, events)
	if !doRefresh {
		t.Error("expected do refresh to be true")
	}

	if len(sess.Values) != 2 {
		t.Error("expected only 2 keys to be set")
	}

	if _, found := sess.Values["key1"]; !found {
		t.Error("cant find key1")
	}
	if _, found := sess.Values["key3"]; !found {
		t.Error("cant find key3")
	}
}
