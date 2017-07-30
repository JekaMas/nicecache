package example

import "testing"

func TestGetKeyHash(t *testing.T) {
	if getHashTestValue([]byte{1}) != getHashTestValue([]byte{1}) {
		t.Fatal("keys dont mapped into the same hash")
	}
}
