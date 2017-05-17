package nicecache

import (
	"testing"
)

var testN int

func Benchmark_Cache_LogHash(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testN = LogHash(i)
	}
}
