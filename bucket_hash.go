package nicecache

func getBucketIdx(h uint64) int {
	return int(h % indexBuckets)
}