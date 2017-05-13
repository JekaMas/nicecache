package nicecache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/sync/syncmap"
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

	index       *syncmap.Map // map[uint64][2]int // map[hashedKey][expiredTime, valueIndexInArray]
	freeIndexes *syncmap.Map // map[int]struct{}
	freeCount   *int32

	onClearing      *int32
	onClearingMutex sync.RWMutex
}

// TODO: добавить логер, метрику в виде определяемых интерфейсов
func NewNiceCache() *Cache {
	freeIndexes := syncmap.Map{} //make(map[int]struct{}, cacheSize)
	for i := 0; i < cacheSize; i++ {
		freeIndexes.Store(i, struct{}{})
	}

	n := int32(cacheSize)
	freeCount := &n

	onClearing := int32(0)
	return &Cache{
		c:           [cacheSize]*TestValue{},
		index:       &syncmap.Map{}, //make(map[uint64][2]int, cacheSize),
		freeIndexes: &freeIndexes,   //freeIndexes,
		freeCount:   freeCount,
		onClearing:  &onClearing,
	}
}

// TODO отпрофилировать под нагрузкой - отдельно исследовать на блокировки
// TODO возможно принимать не duration для expireSeconds а время, когда истечет кэш - по аналогии с context
func (c *Cache) Set(key []byte, value *TestValue, expireSeconds int) error {
	//TODO: сделать в генераторе отдельный параметр, копировать ли получаемое значение. Если он выставлен, то делать копию получаемого значения.

	h := getHash(key)
	resIntf, ok := c.index.Load(h)
	var res [2]int
	if resIntf != nil {
		res = resIntf.([2]int)
	}

	if !ok {
		res[valueIndex] = c.popFreeIndex()
	}

	now := int(time.Now().Unix())
	res[expiredIndex] = now + expireSeconds

	c.index.Store(h, res)

	c.c[res[valueIndex]] = value
	return nil
}
/*
func (c *Cache) Set(key []byte, value TestValue, expireSeconds int) error {
	//TODO: сделать в генераторе отдельный параметр, копировать ли получаемое значение. Если он выставлен, то делать копию получаемого значения.

	h := getHash(key)
	res, ok := c.index.Load(h)
	_ = res

	var freeIdx int
	now := int(time.Now().Unix())

	if !ok {
		freeIdx = c.popFreeIndex()
	} else {
		//freeIdx = res.([2]int)[valueIndex]
		freeIdx = 1
	}

	c.index.Store(h, [2]int{now + expireSeconds, freeIdx})

	c.c[freeIdx] = value
	return nil
}
*/

func (c *Cache) Get(key []byte) (*TestValue, error) {
	h := getHash(key)
	resIntf, ok := c.index.Load(h)
	if !ok {
		return nil, NotFoundError
	}

	now := int(time.Now().Unix())
	res := resIntf.([2]int)
	valueIdx := res[valueIndex]

	if (res[expiredIndex] - now) <= 0 {
		c.delete(h, valueIdx)
		return nil, NotFoundError
	}

	return c.c[valueIdx], nil
}

func (c *Cache) Delete(key []byte) {
	h := getHash(key)
	res, ok := c.index.Load(h)
	if !ok {
		return
	}

	c.index.Delete(h)

	c.pushFreeIndex(res.([2]int)[valueIndex])
}

func (c *Cache) delete(h uint64, valueIdx int) {
	res, ok := c.index.Load(h)
	if !ok {
		return
	}

	c.index.Delete(h)

	c.pushFreeIndex(res.([2]int)[valueIndex])
}

// FIXME Check locks distribution
func (c *Cache) popFreeIndex() int {
	if atomic.LoadInt32(c.freeCount) == 0 {
		// Если индексы иссякли, то считаем свободными процент от записей.
		// TODO заменить на lru?
		// TODO: подумать, что делать с истекшим ttl - надо высвобождать хотя бы частично эти записи. возможно с лимитом времени на gc

		// все горутины берут значение onClearing, но только одна горутина увеличит c.onClearing
		onClearing := atomic.LoadInt32(c.onClearing)
		c.onClearingMutex.Lock()
		if atomic.LoadInt32(c.onClearing) != onClearing {
			goto cleaningDone
		}

		i := 0
		c.index.Range(func(k, res interface{}) bool {
			c.freeIndexes.Store(res.([2]int)[valueIndex], struct{}{})
			c.index.Delete(k)

			i++
			if i >= freeBatchSize {
				return false
			}
			return true
		})

		atomic.AddInt32(c.onClearing, 1)
		c.onClearingMutex.Unlock()

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
	}
cleaningDone:

	idx := -1
	c.freeIndexes.Range(func(k, res interface{}) bool {
		idx = k.(int)
		c.freeIndexes.Delete(k)
		atomic.AddInt32(c.freeCount, -1)

		return false
	})


	if idx == -1 {
		//DEBUG
		panic(fmt.Sprintf("shit happens: %v", c.freeIndexes))
	}

	return idx
}

func (c *Cache) pushFreeIndex(key int) {
	c.freeIndexes.Store(key, struct{}{})
	atomic.AddInt32(c.freeCount, 1)
}

func (c *Cache) Flush() {
	for i := 0; i < cacheSize; i++ {
		c.freeIndexes.Store(i, struct{}{})
	}
	atomic.StoreInt32(c.freeCount, cacheSize)
	c.index = &syncmap.Map{}
}

func (c *Cache) Len() int {
	return cacheSize - int(atomic.LoadInt32(c.freeCount))
}
