package nicecache

import (
	"strconv"
	"testing"
)

var bucketHashTest int

func Benchmark_BucketHash(b *testing.B) {
	b.SkipNow()
	keys := make([][]byte, *repeats)
	hs := make([]uint64, *repeats)
	for i := 0; i < *repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
		hs[i] = getHash(keys[i])
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bucketHashTest = getBucketIDs(hs[i%*repeats])
	}
}

func Benchmark_BucketHash_Naive(b *testing.B) {
	b.SkipNow()
	keys := make([][]byte, *repeats)
	hs := make([]uint64, *repeats)
	for i := 0; i < *repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
		hs[i] = getHash(keys[i])
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bucketHashTest = int(hs[i%*repeats] % indexBuckets)
	}
}
