package nicecache

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	bucketsBin       = 4 // TODO get this from params in generator
	buckets          = (1 << bucketsBin) - 1
	cacheSize        = 1024 * 1024 * 10 // TODO get this from params in generator
	freeBatchPercent = 10               // TODO: tune // TODO get this from params in generator
	expiredIndex     = 0
	valueIndex       = 1

	// TODO get this from params in generator
	alpha              = 1 // Percent increasing speed of freeBatchSize
	maxFreeRatePercent = 33
)

//TODO generate as const
var freeBatchSize int = (cacheSize * freeBatchPercent) / 100

type index struct {
	*sync.RWMutex
	m map[uint64][2]int // map[hashedKey][expiredTime, valueIndexInArray]
}

// TODO: надо обеспечить в генераторе выбор для структур или ссылок на структуры (или и то и то) генерируется кэш
type Cache struct {
	// TODO сделать дополнительное разбиение кэша на buckets в зависимости от размера cacheSize
	c [cacheSize]TestValue // Preallocated storage

	// Разбить на массив [buckets]{sync.RWMutex, index map[uint64][2]int} разбивать по последним 4 битам uint64
	idx [buckets]index

	freeIndexMutex sync.RWMutex
	freeIndexes    map[int]struct{}
	freeCount      *int32
}

// TODO: добавить логер, метрику в виде определяемых интерфейсов
func NewNiceCache() *Cache {
	freeIndexes := make(map[int]struct{}, cacheSize)
	for i := 0; i < cacheSize; i++ {
		freeIndexes[i] = struct{}{}
	}

	n := int32(cacheSize)
	freeCount := &n

	indexes := [buckets]index{}
	for i := 0; i < buckets; i++ {
		mutex := sync.RWMutex{}
		indexes[i] = index{
			&mutex,
			make(map[uint64][2]int, cacheSize),
		}
	}

	return &Cache{
		c:           [cacheSize]TestValue{},
		idx:         indexes,
		freeIndexes: freeIndexes,
		freeCount:   freeCount,
	}
}

// TODO отпрофилировать под нагрузкой - отдельно исследовать на блокировки
// TODO возможно принимать не duration для expireSeconds а время, когда истечет кэш - по аналогии с context
func (c *Cache) Set(key []byte, value TestValue, expireSeconds int) error {
	hashes := getHash(key)
	bucket := c.idx[hashes[bucketIndex]]

	bucket.RLock()
	res, ok := bucket.m[hashes[hashIndex]]
	bucket.RUnlock()

	var freeIdx int
	//TODO: учитывать, что ноды могут быть в разных часовых поясах
	now := int(time.Now().Unix())

	if !ok {
		freeIdx = c.popFreeIndex(hashes[bucketIndex])
	} else {
		freeIdx = res[valueIndex]
	}

	bucket.Lock()
	bucket.m[hashes[hashIndex]] = [2]int{now + expireSeconds, freeIdx}
	bucket.Unlock()

	c.c[freeIdx] = value
	return nil
}

//TODO сделать нормальные ошибки
var NotFoundError = errors.New("key not found")

func (c *Cache) Get(key []byte) (TestValue, error) {
	hashes := getHash(key)
	bucket := c.idx[hashes[bucketIndex]]

	bucket.RLock()
	res, ok := bucket.m[hashes[hashIndex]]
	bucket.RUnlock()

	if !ok {
		return TestValue{}, NotFoundError
	}

	now := int(time.Now().Unix())
	valueIdx := res[valueIndex]

	if (res[expiredIndex] - now) <= 0 {
		c.delete(hashes)
		return TestValue{}, NotFoundError
	}

	return c.c[valueIdx], nil
}

func (c *Cache) Delete(key []byte) {
	hashes := getHash(key)
	bucket := c.idx[hashes[bucketIndex]]

	bucket.RLock()
	res, ok := bucket.m[hashes[hashIndex]]
	bucket.RUnlock()

	bucket.Lock()
	delete(bucket.m, hashes[hashIndex])
	bucket.Unlock()

	if !ok {
		return
	}

	c.pushFreeIndex(res[valueIndex])
}

func (c *Cache) delete(hashes [2]uint64) {
	bucket := c.idx[hashes[bucketIndex]]

	bucket.RLock()
	res, ok := bucket.m[hashes[hashIndex]]
	bucket.RUnlock()

	bucket.Lock()
	delete(bucket.m, hashes[hashIndex])
	bucket.Unlock()

	if !ok {
		return
	}

	c.pushFreeIndex(res[valueIndex])
}

// FIXME Check locks distribution
func (c *Cache) popFreeIndex(bucketIdx uint64) int {
	bucket := c.idx[bucketIdx]

	c.freeIndexMutex.RLock()
	freeIndexesCount := atomic.LoadInt32(c.freeCount)
	c.freeIndexMutex.RUnlock()
	if freeIndexesCount == 0 {
		// Если индексы иссякли, то считаем свободными процент от записей.
		// TODO заменить на lru?
		// TODO запускать в горутине? Сделать логику зависимой от размера кэша?

		i := 0
		indexToFree := make([]int, freeBatchSize)
		bucket.Lock()
		for h, res := range bucket.m {
			indexToFree[i] = res[valueIndex]
			delete(bucket.m, h)

			i++
			if i >= freeBatchSize {
				break
			}
		}
		bucket.Unlock()

		c.freeIndexMutex.Lock()
		for _, idxToFree := range indexToFree {
			c.freeIndexes[idxToFree] = struct{}{}
		}
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

	c.freeIndexMutex.Lock()
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
	c.freeIndexMutex.Lock()
	for i := 0; i < cacheSize; i++ {
		c.freeIndexes[i] = struct{}{}
	}
	atomic.StoreInt32(c.freeCount, cacheSize)
	c.freeIndexMutex.Unlock()

	for idx, bucket := range c.idx {
		mutex := sync.RWMutex{}
		bucket.Lock()
		c.idx[idx] =index{
			&mutex,
			make(map[uint64][2]int, cacheSize),
		}
		bucket.Unlock()
	}
}

func (c *Cache) Len() int {
	return cacheSize - int(atomic.LoadInt32(c.freeCount))
}

//TODO batch get, set, delete ?
