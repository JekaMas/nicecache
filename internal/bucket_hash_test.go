package nicecache

import "testing"

func TestGetBucketIDs(t *testing.T) {
	if getBucketIDs(1) != getBucketIDs(1) {
		t.Fatal("IDs dont mapped into the same bucket")
	}

}
