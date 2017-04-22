package nicecache

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	cacheSize        = 1024 * 1024 * 10
	freeBatchPercent = 10 // TODO: tune
	expiredIndex     = 0
	valueIndex       = 1

	alpha              = 1 // Percent increasing speed of freeBatchSize
	maxFreeRatePercent = 33
)

//TODO generate as const
var freeBatchSize int = (cacheSize * freeBatchPercent) / 100

// TODO: надо обеспечить в генераторе выбор для структур или ссылок на структуры (или и то и то) генерируется кэш
type Cache struct {
	// TODO сделать дополнительное разбиение кэша на buckets в зависимости от размера cacheSize
	c [cacheSize]TestValue // Preallocated storage

	sync.RWMutex
	index map[uint64][2]int // map[hashedKey][expiredTime, valueIndexInArray]

	freeIndexMutex sync.RWMutex
	freeIndexes    map[int]struct{}
	freeCount      *int32
}

func NewNiceCache() *Cache {
	freeIndexes := make(map[int]struct{}, cacheSize)
	for i := 0; i < cacheSize; i++ {
		freeIndexes[i] = struct{}{}
	}

	n := int32(cacheSize)
	freeCount := &n
	return &Cache{
		c:           [cacheSize]TestValue{},
		index:       make(map[uint64][2]int, cacheSize),
		freeIndexes: freeIndexes,
		freeCount:   freeCount,
	}
}

// TODO отпрофилировать под нагрузкой - отдельно исследовать на блокировки
// TODO возможно принимать не duration для expireSeconds а время, когда истечет кэш - по аналогии с context
func (c *Cache) Set(key []byte, value TestValue, expireSeconds int) error {
	h := getHash(key)
	c.RLock()
	res, ok := c.index[h]
	c.RUnlock()

	var freeIdx int
	//TODO: учитывать, что ноды могут быть в разных часовых поясах
	now := int(time.Now().Unix())

	if !ok {
		freeIdx = c.popFreeIndex()
	} else {
		freeIdx = res[valueIndex]
	}

	c.Lock()
	c.index[h] = [2]int{now + expireSeconds, freeIdx}
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
		c.delete(h, valueIdx)
		return TestValue{}, NotFoundError
	}

	return c.c[valueIdx], nil
}

func (c *Cache) Delete(key []byte) {
	h := getHash(key)
	c.RLock()
	res, ok := c.index[h]
	c.RUnlock()

	c.Lock()
	delete(c.index, h)
	c.Unlock()

	if !ok {
		return
	}

	c.pushFreeIndex(res[valueIndex])
}

func (c *Cache) delete(h uint64, valueIdx int) {
	c.RLock()
	res, ok := c.index[h]
	c.RUnlock()

	c.Lock()
	delete(c.index, h)
	c.Unlock()

	if !ok {
		return
	}

	c.pushFreeIndex(res[valueIndex])
}

// FIXME Check locks distribution
func (c *Cache) popFreeIndex() int {
	c.freeIndexMutex.Lock()
	if atomic.LoadInt32(c.freeCount) == 0 {
		// Если индексы иссякли, то считаем свободными процент от записей.
		// TODO заменить на lru?
		// TODO запускать в горутине? Сделать логику зависимой от размера кэша?

		i := 0
		c.Lock()
		for h, res := range c.index {
			c.freeIndexes[res[valueIndex]] = struct{}{}
			delete(c.index, h)

			i++
			if i >= freeBatchSize {
				break
			}
		}
		c.Unlock()

		atomic.StoreInt32(c.freeCount, int32(i))
		c.freeIndexMutex.Unlock()

		// Рост freeBatchSize прогрессивный
		var freeBatchSizeDelta int = freeBatchSize * alpha / 100
		if freeBatchSizeDelta < 1 {
			freeBatchSizeDelta = 1
		}

		freeBatchSize += freeBatchSizeDelta
		if freeBatchSize > (cacheSize*maxFreeRatePercent)/100 {
			freeBatchSize = (cacheSize * maxFreeRatePercent) / 100
		}
	}

	for idx := range c.freeIndexes {
		delete(c.freeIndexes, idx)
		atomic.AddInt32(c.freeCount, -1)
		c.freeIndexMutex.Unlock()

		return idx
	}
	c.freeIndexMutex.Unlock()

	//DEBUG
	panic(fmt.Sprintf("shit happens: %v", c.freeIndexes))

	return 0
}

func (c *Cache) pushFreeIndex(key int) {
	c.freeIndexMutex.Lock()
	c.freeIndexes[key] = struct{}{}
	atomic.AddInt32(c.freeCount, 1)
	c.freeIndexMutex.Unlock()

}

func (c *Cache) Flush() {
	c.Lock()

	c.freeIndexMutex.Lock()
	for i := 0; i < cacheSize; i++ {
		c.freeIndexes[i] = struct{}{}
	}
	atomic.StoreInt32(c.freeCount, cacheSize)
	c.freeIndexMutex.Unlock()

	c.index = make(map[uint64][2]int, cacheSize)
	c.Unlock()
}

func (c *Cache) Len() int {
	//DEBUG - return this in prod
	//return cacheSize - int(atomic.LoadInt32(c.freeCount))

	//DEBUG
	c.RLock()
	len := len(c.index)
	c.RUnlock()
	return len
}

//TODO batch get, set, delete ?
