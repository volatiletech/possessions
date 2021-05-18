package possessions

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Session gets strings
type Session interface {
	Get(key string) (value string, hasKey bool)
}

// session holds the session value and the flash messages key/value mapping
type session struct {
	ID string
	// value is the session value stored as a json encoded string
	Values map[string]string
}

// Get a key
func (s session) Get(key string) (string, bool) {
	str, ok := s.Values[key]
	return str, ok
}

// Storer provides methods to retrieve, add and delete sessions.
type Storer interface {
	// All returns all keys in the store
	All(ctx context.Context) (keys []string, err error)
	Get(ctx context.Context, key string) (value string, err error)
	Set(ctx context.Context, key, value string) error
	Del(ctx context.Context, key string) error
	ResetExpiry(ctx context.Context, key string) error
}

// EventKind of session mutation
type EventKind int

const (
	// EventSet sets a key-value pair
	EventSet EventKind = iota
	// EventDel removes a key
	EventDel
	// EventDelAll means you should delete EVERY key-value pair from
	// the client state - though a whitelist of keys that should not be deleted
	// will be set in Keys
	EventDelAll
	// EventRefresh should refresh the TTL if any on the session
	EventRefresh
	// Deletes the client state
	EventDelClientState
)

// Event represents an operation on a session
type Event struct {
	Kind EventKind
	Key  string
	Val  string
	Keys []string
}

// Overseer of session cookies
type Overseer interface {
	// ReadState should return a map like structure allowing it to look up
	// any values in the current session, or any cookie in the request
	ReadState(*http.Request) (Session, error)
	// WriteState can sometimes be called with a nil ClientState in the event
	// that no ClientState was read in from LoadClientState
	WriteState(context.Context, http.ResponseWriter, Session, []Event) error
}

// timer interface is used to mock the test harness for disk and memory storers
type timer interface {
	Stop() bool
	Reset(time.Duration) bool
}

type noSessionInterface interface {
	NoSession()
}
type noMapKeyInterface interface {
	NoMapKey()
}

type errNoSession struct{}
type errNoMapKey struct{}

func (errNoSession) NoSession() {}
func (errNoMapKey) NoMapKey()   {}

func (errNoSession) Error() string {
	return "session does not exist"
}
func (errNoMapKey) Error() string {
	return "session map key does not exist"
}

// IsNoSessionError checks an error to see if it means that there was no session
func IsNoSessionError(err error) bool {
	_, ok := err.(noSessionInterface)
	if ok {
		return ok
	}

	_, ok = errors.Cause(err).(noSessionInterface)
	return ok
}

// IsNoMapKeyError checks an error to see if it means that there was
// no session map key
func IsNoMapKeyError(err error) bool {
	_, ok := err.(noMapKeyInterface)
	if ok {
		return ok
	}

	_, ok = errors.Cause(err).(noMapKeyInterface)
	return ok
}

// timerTestHarness allows us to control the timer channels manually in the
// disk and memory storer tests so that we can trigger cleans at will
var timerTestHarness = func(d time.Duration) (timer, <-chan time.Time) {
	t := time.NewTimer(d)
	return t, t.C
}

// validKey returns true if the session key is a valid UUIDv4 format:
// 8chars-4chars-4chars-4chars-12chars (chars are a-f 0-9)
// Example: a668b3bb-0cf1-4627-8cd4-7f62d09ebad6
func validKey(key string) bool {
	// UUIDv4's are 36 chars (16 bytes not including dashes)
	if len(key) != 36 {
		return false
	}

	// 0 indexed dash positions
	dashPos := []int{8, 13, 18, 23}
	for i := 0; i < len(key); i++ {
		atDashPos := false
		for _, pos := range dashPos {
			if i == pos {
				atDashPos = true
				break
			}
		}

		if atDashPos == true {
			if key[i] != '-' {
				return false
			}
			// continue the loop if dash is found
			continue
		}

		// if not a dash, make sure char is a-f or 0-9
		// 48 == '0', 57 == '9', 97 == 'a', 102 == 'f'
		if key[i] < 48 || (key[i] > 57 && key[i] < 97) || key[i] > 102 {
			return false
		}
	}

	return true
}

// Get a session string value
func Get(ctx context.Context, key string) (string, bool) {
	return get(ctx, key)
}

// GetObj a session json encoded string and decode it into obj. Use the
// IsNoMapKeyError to determine if the value was found or not.
func GetObj(ctx context.Context, key string, obj interface{}) error {
	encodedString, ok := get(ctx, key)
	if !ok {
		return errNoMapKey{}
	}

	err := json.Unmarshal([]byte(encodedString), obj)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal session key-value")
	}

	return nil
}

// Set a session-value string
func Set(w http.ResponseWriter, key, value string) {
	set(w, key, value)
}

// SetObj marshals the value to a json string and sets it in the session
func SetObj(w http.ResponseWriter, key string, obj interface{}) error {
	value, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	set(w, key, string(value))
	return nil
}

func get(ctx context.Context, key string) (string, bool) {
	cached := ctx.Value(CTXKeyPossessions{})
	if cached == nil {
		return "", false
	}

	sess, ok := cached.(Session)
	if !ok {
		panic("cached session value does not conform to possesions.Session interface")
	}

	return sess.Get(key)
}

func set(w http.ResponseWriter, key, value string) {
	pw := getResponseWriter(w)

	pw.events = append(pw.events, Event{
		Kind: EventSet,
		Key:  key,
		Val:  value,
	})
}

// Del a session key
func Del(w http.ResponseWriter, key string) {
	pw := getResponseWriter(w)

	pw.events = append(pw.events, Event{
		Kind: EventDel,
		Key:  key,
	})
}

// DelAll delete all keys except for a whitelist
func DelAll(w http.ResponseWriter, whitelist []string) {
	pw := getResponseWriter(w)

	pw.events = append(pw.events, Event{
		Kind: EventDelAll,
		Keys: whitelist,
	})
}

// Refresh a session's ttl
func Refresh(w http.ResponseWriter) {
	pw := getResponseWriter(w)

	pw.events = append(pw.events, Event{
		Kind: EventRefresh,
	})
}

// AddFlash adds a flash message to the session. Typically read and removed
// on the next request.
func AddFlash(w http.ResponseWriter, key string, value string) {
	Set(w, key, value)
}

// AddFlashObj adds a flash message to the session using an object that's
// marshalled into JSON
func AddFlashObj(w http.ResponseWriter, key string, obj interface{}) error {
	return SetObj(w, key, obj)
}

// GetFlash reads a flash message from the request and deletes it using the
// responsewriter.
func GetFlash(w http.ResponseWriter, ctx context.Context, key string) (string, bool) {
	flash, ok := Get(ctx, key)
	if !ok {
		return "", false
	}
	Del(w, key)

	return flash, true
}

// GetFlashObj reads a json-encoded flash message from the session and
// unmarshals it into obj. Use IsNoMapKeyError to determine if the value was
// found or not.
func GetFlashObj(w http.ResponseWriter, ctx context.Context, key string, obj interface{}) error {
	flash, ok := Get(ctx, key)
	if !ok {
		return errNoMapKey{}
	}
	Del(w, key)

	err := json.Unmarshal([]byte(flash), obj)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal flash key-value string")
	}

	return nil
}

func getResponseWriter(w http.ResponseWriter) *possesionsWriter {
	for {
		if r, ok := w.(*possesionsWriter); ok {
			return r
		}

		u, ok := w.(UnderlyingResponseWriter)
		if !ok {
			panic("http.ResponseWriter was not possessions.responseWriter no posssessions.UnderlyingResponseWriter")
		}

		w = u.UnderlyingResponseWriter()
	}
}
