package mylogin

import (
	"crypto/rand"
	"testing"
)

func TestKeyIsZero(t *testing.T) {
	var key Key
	if !key.IsZero() {
		t.Fatal("should be IsZero")
	}
	key[19] = 2
	if key.IsZero() {
		t.Fatal("should not be IsZero")
	}
	key[0] = 5
	if key.IsZero() {
		t.Fatal("should not be IsZero")
	}
}

func TestKeyNew(t *testing.T) {
	key, err := NewKey(rand.Read)
	if err != nil {
		t.Fatalf("NewKey: %s", err)
	}
	t.Logf("key: %X", key)
	// IsZero() is possible, but very unlikely
	if key.IsZero() {
		t.Fatal("shouldn't be IsZero")
	}
	for i, c := range key {
		if c >= 32 {
			t.Errorf("byte #%d: %d > 31", i, c)
		}
	}
}
