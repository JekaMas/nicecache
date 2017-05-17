package nicecache

import "math"

// fast implementation of `int(math.Log2(float64(n>>10 + 1)))`
var lookupLog = [256]int{}

func init() {
	lookupLog[0] = 0
	for i := 1; i < 256; i++ {
		lookupLog[i] = int(math.Log(float64(i)) / math.Ln2)
	}
}

// Example of using in getting bucket number https://play.golang.org/p/ZHLXHs9FVY
func LogHash(n int) int {
	n = n>>10 + 1
	if n >= 0x1000000 {
		return lookupLog[n>>24] + 24
	}

	if n >= 0x10000 {
		return lookupLog[n>>16] + 16
	}

	if n >= 0x100 {
		return lookupLog[n>>8] + 8
	}

	return lookupLog[n]
}
