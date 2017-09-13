package nicecache

import (
	"runtime"
	"testing"
)

var l *List

func BenchmarkList(t *testing.B) {
	for j:=0; j<t.N; j++ {
		l = New()

		i := uint64(0)
		for i < 10 {
			l.PushFront(i)
			i += 1
		}
	}

	runtime.KeepAlive(l)
}
