package nicecache

import "github.com/opennota/fasthash"

func getHash(item []byte) uint64 {
	const primeSeed = 380383699

	return fasthash.Hash64(primeSeed, item)
}
