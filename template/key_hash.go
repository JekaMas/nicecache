package nicecache

import (
	"github.com/dgryski/go-farm"
)

func getHash(key []byte) uint64 {
	return farm.Hash64(key)
}
