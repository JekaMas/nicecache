package internal

import "testing"

func TestGetKeyHash(t *testing.T) {
	if getHash([]byte{1}) != getHash([]byte{1}) {
		t.Fatal("keys dont mapped into the same hash")
	}
}
