package nicecache

func getBucketIdx(h uint64) int {
	return int(h % indexBuckets)
}

func getLruShardIdx(h int) int {
	return h % lruShards
}