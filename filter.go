package nicecache

import (
	//"github.com/opennota/fasthash"
	"github.com/dgryski/go-farm"
)

// TODO: generate new big Prime
const primeSeed = 380383699 // some big prime

/*
func getHash(item []byte) uint64 {
	return fasthash.Hash64(primeSeed, item)
}
*/

func getHash(item []byte) uint64 {
	return farm.Hash64(item)
}
