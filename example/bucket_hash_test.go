package example

import "testing"

func TestGetBucketIDs(t *testing.T) {
	if getBucketIDsTestValue(1) != getBucketIDsTestValue(1) {
		t.Fatal("IDs dont mapped into the same bucket")
	}

}
