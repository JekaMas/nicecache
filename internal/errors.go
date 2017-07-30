package nicecache

type cacheError string

func (e cacheError) Error() string { return string(e) }

const (
	NotFoundError = cacheError("key not found")
	NilValueError = cacheError("value pointer shouldn`t be nil")
	CloseError    = cacheError("cache has been closed")

	chunksNegativeSliceSize = cacheError("sliceLen should be non-negative")
	chunksNegativeSize = cacheError("chunkSize should be positive")
)
