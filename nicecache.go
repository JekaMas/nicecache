package nicecache

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	cacheSize        = 1024 * 1024 * 10
	freeBatchPercent = 1
	expiredIndex     = 0
	valueIndex       = 1

	alpha              = 1 // Percent increasing speed of freeBatchSize
	maxFreeRatePercent = 33
)

var freeBatchSize int = (cacheSize * freeBatchPercent) / 100

// TODO: надо обеспечить в генераторе выбор для структур или ссылок на структуры (или и то и то) генерируется кэш
type Cache struct {
	c [cacheSize]*TestValue // Preallocated storage

	sync.RWMutex
	index map[uint64][2]int // map[hashedKey][expiredTime, valueIndexInArray]

	freeIndexMutex sync.RWMutex
	freeIndexes    map[int]struct{}
	freeCount      *int32
	freeIndexCh    chan int

	onClearing      *int32
	startClearingCh chan struct{}

	stop chan struct{}
}

// TODO: добавить логер, метрику в виде определяемых интерфейсов
func NewNiceCache() *Cache {
	freeIndexes := make(map[int]struct{}, cacheSize)
	for i := 0; i < cacheSize; i++ {
		freeIndexes[i] = struct{}{}
	}

	n := int32(cacheSize)
	freeCount := &n

	onClearing := int32(0)
	c := &Cache{
		c:               [cacheSize]*TestValue{},
		index:           make(map[uint64][2]int, cacheSize),
		freeIndexes:     freeIndexes,
		freeCount:       freeCount,
		onClearing:      &onClearing,
		freeIndexCh:     make(chan int, (cacheSize*maxFreeRatePercent)/100),
		startClearingCh: make(chan struct{}, 1),
		stop:            make(chan struct{}, 1), //TODO: close it when stop Cache
	}

	go c.clearCache(c.startClearingCh, c.freeIndexCh)

	return c
}

// TODO отпрофилировать под нагрузкой - отдельно исследовать на блокировки
func (c *Cache) Set(key []byte, value *TestValue, expireSeconds int) error {
	h := getHash(key)
	c.RLock()
	res, ok := c.index[h]
	c.RUnlock()

	var freeIdx int
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

func (c *Cache) Get(key []byte) (*TestValue, error) {
	h := getHash(key)
	c.RLock()
	res, ok := c.index[h]
	c.RUnlock()

	if !ok {
		return nil, NotFoundError
	}

	now := int(time.Now().Unix())
	valueIdx := res[valueIndex]

	if (res[expiredIndex] - now) <= 0 {
		c.delete(h, valueIdx)
		return nil, NotFoundError
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
	// Если индексы иссякли, то считаем свободными процент от записей.
	if atomic.LoadInt32(c.freeCount) == 0 {
		if atomic.CompareAndSwapInt32(c.onClearing, 0, 1) {
			c.startClearingCh <- struct{}{}
		}
	}

	var idx int
	if atomic.LoadInt32(c.freeCount) == 0 {
		idx = <-c.freeIndexCh
	} else {
		c.freeIndexMutex.RLock()
		for idx = range c.freeIndexes {
			break
		}
		c.freeIndexMutex.RUnlock()
	}

	c.freeIndexMutex.Lock()
	delete(c.freeIndexes, idx)
	c.freeIndexMutex.Unlock()

	atomic.AddInt32(c.freeCount, -1)

	return idx
}

func (c *Cache) clearCache(startClearingCh chan struct{}, freeIndexCh chan int) {
	for {
		select {
		case <-startClearingCh:
			// TODO заменить на lru?
			// TODO: подумать, что делать с истекшим ttl - надо высвобождать хотя бы частично эти записи. возможно с лимитом времени на gc
			i := 0
			c.freeIndexMutex.Lock()
			c.Lock()
			for h, res := range c.index {
				freeIndexCh <- res[valueIndex]
				c.freeIndexes[res[valueIndex]] = struct{}{}
				delete(c.index, h)

				i++
				if i >= freeBatchSize {
					break
				}
			}

			c.Unlock()
			c.freeIndexMutex.Unlock()

			atomic.StoreInt32(c.freeCount, int32(i))

			// Increase freeBatchSize progressive
			var freeBatchSizeDelta int = freeBatchSize * alpha / 100
			if freeBatchSizeDelta < 1 {
				freeBatchSizeDelta = 1
			}

			freeBatchSize += freeBatchSizeDelta
			if freeBatchSize > (cacheSize*maxFreeRatePercent)/100 {
				freeBatchSize = (cacheSize * maxFreeRatePercent) / 100
			}

			atomic.StoreInt32(c.onClearing, 0)
		case <-c.stop:
			return
		}
	}
}

func (c *Cache) pushFreeIndex(key int) {
	c.freeIndexMutex.Lock()
	c.freeIndexes[key] = struct{}{}
	c.freeIndexMutex.Unlock()

	atomic.AddInt32(c.freeCount, 1)
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
	return cacheSize - int(atomic.LoadInt32(c.freeCount))
}
