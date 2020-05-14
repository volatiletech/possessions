package possessions

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
)

func TestRedisStorerNew(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping long test")
	}

	r := redis.Options{
		Password: "test",
	}
	storer, err := NewRedisStorer(r, 2)
	if err != nil {
		t.Error(err)
	}

	if storer.maxAge != 2 {
		t.Error("expected max age to be 2")
	}

	if storer.client == nil {
		t.Error("Expected client to be created")
	}
}

func TestRedisStorerNewDefault(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping long test")
	}

	storer, err := NewDefaultRedisStorer("", "", 13)
	if err != nil {
		t.Error(err)
	}

	if storer.client == nil {
		t.Error("Expected client to be created")
	}

	if storer.maxAge != time.Hour*24*2 {
		t.Error("expected max age to be 2 days")
	}
}

func TestRedisStorerAll(t *testing.T) {
	t.Parallel()

	s, err := NewDefaultRedisStorer("", "", 13)
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()

	list, err := s.All(ctx)
	if err != nil {
		t.Error("expected no error on empty list")
	}
	if len(list) > 0 {
		t.Error("Expected len 0")
	}

	s.Set(ctx, "hi", "hello")
	s.Set(ctx, "yo", "friend")

	list, err = s.All(ctx)
	if err != nil {
		t.Error(err)
	}
	if len(list) != 2 {
		t.Errorf("Expected len 2, got %d", len(list))
	}
	if (list[0] != "hi" && list[0] != "yo") || list[0] == list[1] {
		t.Errorf("Expected list[0] to be %q or %q, got %q", "yo", "hi", list[0])
	}
	if (list[1] != "yo" && list[1] != "hi") || list[1] == list[0] {
		t.Errorf("Expected list[1] to be %q or %q, got %q", "hi", "yo", list[1])
	}

	// Cleanup
	s.Del(ctx, "hi")
	s.Del(ctx, "yo")
}

func TestRedisStorerGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	storer, err := NewDefaultRedisStorer("", "", 13)
	if err != nil {
		t.Error(err)
	}

	testidUUID, err := uuid.NewV4()
	if err != nil {
		t.Error(err)
	}

	testid1 := testidUUID.String()

	val, err := storer.Get(context.Background(), testid1)
	if !IsNoSessionError(err) {
		t.Errorf("Expected ErrNoSession, got: %v", err)
	}

	storer.Set(context.Background(), testid1, "banana")

	val, err = storer.Get(context.Background(), testid1)
	if err != nil {
		t.Error(err)
	}
	if val != "banana" {
		t.Errorf("Expected %q, got %q", "banana", val)
	}

	// Cleanup
	storer.Del(context.Background(), testid1)
}

func TestRedisStorerSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	storer, err := NewDefaultRedisStorer("", "", 13)
	if err != nil {
		t.Error(err)
	}

	testid1 := uuid.Must(uuid.NewV4()).String()
	testid2 := uuid.Must(uuid.NewV4()).String()

	ctx := context.Background()

	storer.Set(ctx, testid1, "hello")
	storer.Set(ctx, testid1, "whatsup")
	storer.Set(ctx, testid2, "friend")

	val, err := storer.Get(ctx, testid1)
	if err != nil {
		t.Fatal(err)
	}
	if val != "whatsup" {
		t.Errorf("Expected %q, got %q", "whatsup", val)
	}

	val, err = storer.Get(ctx, testid2)
	if err != nil {
		t.Error(err)
	}
	if val != "friend" {
		t.Errorf("Expected %q, got %q", "friend", val)
	}

	// Cleanup
	storer.Del(ctx, testid1)
	storer.Del(ctx, testid2)
}

func TestRedisStorerDel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	storer, err := NewDefaultRedisStorer("", "", 13)
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()

	storer.Set(ctx, "hi", "hello")
	storer.Set(ctx, "hi", "whatsup")
	storer.Set(ctx, "yo", "friend")

	err = storer.Del(ctx, "hi")
	if err != nil {
		t.Error(err)
	}

	_, err = storer.Get(ctx, "hi")
	if err == nil {
		t.Errorf("Expected get hi to fail")
	}

	// Cleanup
	storer.Del(ctx, "hi")
	storer.Del(ctx, "yo")
}

func TestRedisStorerResetExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	storer, err := NewDefaultRedisStorer("", "", 13)
	if err != nil {
		t.Error(err)
	}
	// Set maxage duration to 1 hour
	storer.maxAge = time.Hour * 1

	ctx := context.Background()

	err = storer.Set(ctx, "test", "test1")
	if err != nil {
		t.Error(err)
	}

	oldDur, err := storer.client.TTL(ctx, "test").Result()
	if err != nil {
		t.Error(err)
	}

	// Make sure the duration is roughly 1 hour and no greater
	if oldDur > time.Hour*1 || oldDur < time.Minute*59 {
		t.Errorf("expected TTL in Redis to be set to 1 hour, but got: %v", oldDur.String())
	}

	// Adjust the duration to something else so we can ensure it was reset
	storer.maxAge = time.Hour * 24

	err = storer.ResetExpiry(ctx, "test")
	if err != nil {
		t.Error(err)
	}

	newDur, err := storer.client.TTL(ctx, "test").Result()
	if err != nil {
		t.Error(err)
	}

	// Make sure the new reset duration is roughly 1 day and no less
	if newDur > time.Hour*24 || newDur < time.Hour*23 {
		t.Errorf("expected TTL in Redis to be set to 24 hours, but got: %v", newDur.String())
	}

	// Cleanup
	storer.Del(ctx, "test")
}
