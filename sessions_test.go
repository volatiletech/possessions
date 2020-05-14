package possessions

import (
	"context"
	"testing"
)

func TestSetAndSetObjAndDelAndDelAllAndRefreshAndFlashAndFlashObj(t *testing.T) {
	t.Parallel()

	w := newResponseWriter(context.Background(), nil, nil)

	Set(w, "key1", "value1")
	SetObj(w, "key2", "value2")
	AddFlash(w, "key3", "value3")
	AddFlashObj(w, "key4", "value4")
	Del(w, "key5")
	DelAll(w, []string{"key6"})
	Refresh(w)

	if len(w.events) != 7 {
		t.Error("expected 7 events, got:", len(w.events))
	}

	if w.events[0].Kind != EventSet || w.events[0].Key != "key1" || w.events[0].Val != "value1" {
		t.Error("expected event set, key1, value1", w.events[0])
	}
	if w.events[1].Kind != EventSet || w.events[1].Key != "key2" || w.events[1].Val != `"value2"` {
		t.Error("expected event set, key2, value2", w.events[1])
	}
	if w.events[2].Kind != EventSet || w.events[2].Key != "flash_key3" || w.events[2].Val != "value3" {
		t.Error("expected event set, key3, value3", w.events[2])
	}
	if w.events[3].Kind != EventSet || w.events[3].Key != "flash_key4" || w.events[3].Val != `"value4"` {
		t.Error("expected event set, key4, value4", w.events[3])
	}
	if w.events[4].Kind != EventDel || w.events[4].Key != "key5" {
		t.Error("expected event del, key5", w.events[4])
	}
	if w.events[5].Kind != EventDelAll || w.events[5].Keys[0] != "key6" {
		t.Error("expected event del all, key6", w.events[5])
	}
	if w.events[6].Kind != EventRefresh {
		t.Error("expected event del refresh")
	}
}

func TestGetAndGetObjAndGetFlashAndGetFlashObj(t *testing.T) {
	t.Parallel()

	uuid := "816a1acb-73aa-4a75-bbeb-f371bdad40e8"

	sess := session{
		ID: uuid,
		Values: map[string]string{
			"key1":       "value1",
			"key2":       `"value2"`,
			"flash_key3": "value3",
			"flash_key4": `"value4"`,
		},
	}
	ctx := context.WithValue(context.Background(), CTXKeyPossessions{}, sess)

	val, ok := Get(ctx, "key1")
	if val != "value1" || !ok {
		t.Error("expected value to be value1, but got:", val)
	}

	var val2 string
	err := GetObj(ctx, "key2", &val2)
	if err != nil {
		t.Error("unable to get key2 object using GetObj")
	}
	if val2 != "value2" {
		t.Error("expected value to be \"value2\", got:", val2)
	}

	w := newResponseWriter(context.Background(), nil, nil)

	val, ok = GetFlash(w, ctx, "key3")
	if val != "value3" || !ok {
		t.Error("expected value to be value3, got:", val)
	}

	var val4 string
	err = GetFlashObj(w, ctx, "key4", &val4)
	if err != nil {
		t.Error("unable to get key4 flash object using GetFlashObj")
	}

	if len(w.events) != 2 {
		t.Error("expected 2 events, got:", len(w.events))
	}
	if w.events[0].Kind != EventDel || w.events[0].Key != "flash_key3" {
		t.Error("expected event set, flash_key3, value3", w.events[0])
	}
	if w.events[1].Kind != EventDel || w.events[1].Key != "flash_key4" {
		t.Error("expected event set, flash_key4, value4", w.events[1])
	}
}
