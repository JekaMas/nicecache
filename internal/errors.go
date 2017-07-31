package internal

type cacheError string

func (e cacheError) Error() string { return string(e) }

const (
	//NotFoundError not found error
	NotFoundError = cacheError("key not found")
	//NilValueError nil pointer given
	NilValueError = cacheError("value pointer shouldn`t be nil")
	//CloseError cache is closed
	CloseError = cacheError("cache has been closed")

	chunksNegativeSliceSize = cacheError("sliceLen should be non-negative")
	chunksNegativeSize      = cacheError("chunkSize should be positive")
)
