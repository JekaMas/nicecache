package nicecache

type cacheError string

func (e cacheError) Error() string { return string(e) }

const NotFoundError = cacheError("key not found")
