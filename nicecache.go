package nicecache

import (
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

	indexMap    *syncmap.Map      // map[uint64][2]int // map[hashedKey][expiredTime, valueIndexInArray]
	index       [cacheSize][2]int // map[uint64][2]int // map[hashedKey][expiredTime, valueIndexInArray]
	freeIndexes *syncmap.Map      // map[int]struct{}
	freeCount   *int32
	freeIndexCh chan int

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
		indexMap:    &syncmap.Map{}, //make(map[uint64][2]int, cacheSize),
		freeIndexes: &freeIndexes,   //freeIndexes,
		freeCount:   freeCount,
		onClearing:  &onClearing,
		freeIndexCh: make(chan int, (cacheSize*maxFreeRatePercent)/100),
	}
}

// TODO отпрофилировать под нагрузкой - отдельно исследовать на блокировки
func (c *Cache) Set(key []byte, value *TestValue, expireSeconds int) error {
	//TODO: сделать в генераторе отдельный параметр, копировать ли получаемое значение. Если он выставлен, то делать копию получаемого значения.
	var resIndex int
	h := getHash(key)

	resIndexIntf, ok := c.indexMap.Load(h)
	if resIndexIntf != nil {
		resIndex = resIndexIntf.(int)
	}

	if !ok {
		resIndex = c.popFreeIndex()
	}

	c.index[resIndex][valueIndex] = resIndex
	now := int(time.Now().Unix())
	c.index[resIndex][expiredIndex] = now + expireSeconds

	c.indexMap.Store(h, resIndex)

	c.c[c.index[resIndex][valueIndex]] = value
	return nil
}

func (c *Cache) Get(key []byte) (*TestValue, error) {
	h := getHash(key)
	resIndexIntf, ok := c.indexMap.Load(h)
	if !ok {
		return nil, NotFoundError
	}

	now := int(time.Now().Unix())
	resIndex := resIndexIntf.(int)
	valueIdx := c.index[resIndex][valueIndex]

	if (c.index[resIndex][expiredIndex] - now) <= 0 {
		c.delete(h, valueIdx)
		return nil, NotFoundError
	}

	return c.c[valueIdx], nil
}

func (c *Cache) Delete(key []byte) {
	h := getHash(key)
	res, ok := c.indexMap.Load(h)
	if !ok {
		return
	}

	c.indexMap.Delete(h)

	c.pushFreeIndex(res.([2]int)[valueIndex])
}

func (c *Cache) delete(h uint64, valueIdx int) {
	resIndexIntf, ok := c.indexMap.Load(h)
	if !ok {
		return
	}

	c.indexMap.Delete(h)
	resIndex := resIndexIntf.(int)
	c.pushFreeIndex(c.index[resIndex][valueIndex])
}

// FIXME Check locks distribution
func (c *Cache) popFreeIndex() int {
	if atomic.LoadInt32(c.freeCount) == 0 {
		go c.clearCache(c.freeIndexCh)
	}

	var idx int
	if atomic.LoadInt32(c.freeCount) == 0 {
		idx = <-c.freeIndexCh
	} else {
		c.freeIndexes.Range(func(k, res interface{}) bool {
			idx = k.(int)
			c.freeIndexes.Delete(idx)
			atomic.AddInt32(c.freeCount, -1)
			return false
		})
	}

	return idx
}

func (c *Cache) clearCache(freeIndexCh chan int) {
	// Если индексы иссякли, то считаем свободными процент от записей.
	// TODO заменить на lru?
	// TODO: подумать, что делать с истекшим ttl - надо высвобождать хотя бы частично эти записи. возможно с лимитом времени на gc

	// все горутины берут значение onClearing, но только одна горутина увеличит c.onClearing
	onClearing := atomic.LoadInt32(c.onClearing)
	c.onClearingMutex.Lock()
	if atomic.LoadInt32(c.onClearing) != onClearing {
		c.onClearingMutex.Unlock()
		return
	}

	i := 0
	c.indexMap.Range(func(k, res interface{}) bool {
		resIndex := res.(int)
		v := c.index[resIndex][valueIndex]
		freeIndexCh <- v
		c.freeIndexes.Store(v, struct{}{})
		c.indexMap.Delete(k)

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

	return
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
	c.indexMap = &syncmap.Map{}
}

func (c *Cache) Len() int {
	return cacheSize - int(atomic.LoadInt32(c.freeCount))
}
