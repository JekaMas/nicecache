package nicecache

import (
	"github.com/dgryski/go-farm"
)

func getHash(item []byte) uint64 {
	return farm.Hash64(item)
}