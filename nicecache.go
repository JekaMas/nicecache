package nicecache

import (
	"errors"
	"sync"
	"time"
)

const (
	cacheSize        = 1024
	freeBatchPercent = 0.1
	freeBatchSize    = 102 //int(cacheSize * float32(freeBatchPercent)) TODO generate
	expiredIndex     = 0
	valueIndex       = 1
)

type Cache struct {
	c [cacheSize]TestValue

	sync.RWMutex
	index map[uint64][2]int

	freeIndexMutex sync.RWMutex
	freeIndexes    map[int]struct{}
}

func NewNiceCache() *Cache {
	freeIndexes := make(map[int]struct{}, cacheSize)
	for i := 0; i < cacheSize; i++ {
		freeIndexes[i] = struct{}{}
	}

	return &Cache{
		c:           [cacheSize]TestValue{},
		index:       make(map[uint64][2]int, cacheSize),
		freeIndexes: freeIndexes,
	}
}

func (c *Cache) Set(key []byte, value TestValue, expireSeconds int) error {
	h := getHash(key)

	//TODO: учитывать, что ноды могут быть в разных часовых поясах
	now := int(time.Now().Unix())
	freeIdx := c.popFreeIndex()

	c.Lock()
	c.index[h] = [2]int{now + expireSeconds, c.popFreeIndex()}
	c.Unlock()

	c.c[freeIdx] = value
	return nil
}

//TODO сделать нормальные ошибки
var NotFoundError = errors.New("key not found")

func (c *Cache) Get(key []byte) (TestValue, error) {
	h := getHash(key)
	c.RLock()
	res, ok := c.index[h]
	c.RUnlock()

	if !ok {
		return TestValue{}, NotFoundError
	}

	now := int(time.Now().Unix())
	valueIdx := res[valueIndex]

	if (res[expiredIndex] - now) <= 0 {
		c.pushFreeIndex(valueIdx)
		return TestValue{}, NotFoundError
	}

	return c.c[valueIdx], nil
}

func (c *Cache) Delete(key []byte) {
	h := getHash(key)
	c.RLock()
	res, ok := c.index[h]
	c.RUnlock()

	if !ok {
		return
	}

	c.pushFreeIndex(res[valueIndex])
}

// FIXME Allocates 2 objects !!!
func (c *Cache) popFreeIndex() int {
	var key int

	c.freeIndexMutex.RLock()
	indexCount := len(c.freeIndexes)
	c.freeIndexMutex.RUnlock()

	if indexCount == 0 {
		// Если индексы иссякли, то считаем свободными процент от записей.
		// TODO заменить на lru?
		c.freeIndexMutex.Lock()
		for i := 0; i < freeBatchSize; i++ {
			c.freeIndexes[i] = struct{}{}
		}
		c.freeIndexMutex.Unlock()
	}

	c.freeIndexMutex.RLock()
	for key = range c.freeIndexes {
		break
	}
	c.freeIndexMutex.RUnlock()

	c.freeIndexMutex.Lock()
	delete(c.freeIndexes, key)
	c.freeIndexMutex.Unlock()

	return key
}

func (c *Cache) pushFreeIndex(key int) {
	c.freeIndexMutex.Lock()
	c.freeIndexes[key] = struct{}{}
	c.freeIndexMutex.Unlock()
}

func (c *Cache) Flush() {
	c.freeIndexMutex.Lock()
	for i := 0; i < cacheSize; i++ {
		c.freeIndexes[i] = struct{}{}
	}
	c.freeIndexMutex.Unlock()
}
