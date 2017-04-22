package nicecache

import "github.com/opennota/fasthash"

const (
	bucketIndex = 0
	hashIndex   = 1
)

func getHash(item []byte) [2]uint64 {
	const primeSeed = 380383699

	h := fasthash.Hash64(primeSeed, item)
	bits := h & bucketsBin
	return [2]uint64{bits, h}
}
