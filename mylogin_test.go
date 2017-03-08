package mylogin

import (
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
