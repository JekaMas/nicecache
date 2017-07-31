package internal

func getBucketIDs(h uint64) int {
	return int(h % indexBuckets)
}
