package possessions

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// StorageOverseer holds cookie related variables and a session storer
type StorageOverseer struct {
	Storer  Storer
	options CookieOptions
}

// NewStorageOverseer returns a new storage overseer
func NewStorageOverseer(opts CookieOptions, storer Storer) *StorageOverseer {
	if len(opts.Name) == 0 {
		panic("cookie name must be provided")
	}

	return &StorageOverseer{
		Storer:  storer,
		options: opts,
	}
}

// ReadState from the request
func (s StorageOverseer) ReadState(r *http.Request) (Session, error) {
	id, err := s.options.getCookieValue(r)
	if err != nil {
		if IsNoSessionError(err) {
			return nil, nil
		}
		return nil, err
	}

	if !validKey(id) {
		return nil, errNoSession{}
	}

	encodedSession, err := s.Storer.Get(r.Context(), id)
	if err != nil {
		return nil, err
	}

	sessValues := make(map[string]string)
	if err = json.Unmarshal([]byte(encodedSession), &sessValues); err != nil {
		return nil, err
	}

	return session{
		ID:     id,
		Values: sessValues,
	}, nil
}

func applyEvents(sessionObj session, evs []Event) (doRefresh bool) {
	for _, ev := range evs {
		switch ev.Kind {
		case EventSet:
			sessionObj.Values[ev.Key] = ev.Val
		case EventDel:
			delete(sessionObj.Values, ev.Key)
		case EventDelAll:
			for k := range sessionObj.Values {
				whitelisted := false
				for _, w := range ev.Keys {
					if k == w {
						whitelisted = true
						break
					}
				}

				if whitelisted {
					continue
				}

				delete(sessionObj.Values, k)
			}
		case EventRefresh:
			doRefresh = true
		}
	}

	return doRefresh
}

// WriteState to the response
func (s StorageOverseer) WriteState(ctx context.Context, w http.ResponseWriter, sess Session, evs []Event) error {
	if len(evs) == 1 && evs[0].Kind == EventDelClientState {
		s.options.deleteCookie(w)
		return nil
	}

	var sessionObj session
	isNew := false
	if sess != nil {
		sessionObj = sess.(session)
	} else {
		isNew = true
		uuidID, err := uuid.NewV4()
		if err != nil {
			return errors.Wrap(err, "failed to create uuid for session")
		}

		sessionObj = session{
			ID:     uuidID.String(),
			Values: make(map[string]string),
		}
	}

	doRefresh := applyEvents(sessionObj, evs) && !isNew

	encodedValues, err := json.Marshal(sessionObj.Values)
	if err != nil {
		return errors.Wrap(err, "failed to marshal session values to json")
	}

	err = s.Storer.Set(ctx, sessionObj.ID, string(encodedValues))
	if err != nil {
		return errors.Wrap(err, "failed to store session values")
	}

	if doRefresh {
		if err = s.Storer.ResetExpiry(ctx, sessionObj.ID); err != nil {
			return errors.Wrap(err, "failed to refresh session")
		}
	}

	if isNew || doRefresh {
		cookie := s.options.makeCookie(sessionObj.ID)
		http.SetCookie(w, cookie)
	}

	return nil
}
