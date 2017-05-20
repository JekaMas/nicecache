package nicecache

func getBucketIdx(h uint64) int {
	return int(h % indexBuckets)
}

func getBucketIntIdx(h int) int {
	return h % indexBuckets
}