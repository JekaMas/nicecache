package nicecache

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	cacheSize    = 1024 * 1024 * 10
	indexBuckets = 100

	freeBatchPercent   = 1
	alpha              = 1 // Percent increasing speed of freeBatchSize
	maxFreeRatePercent = 33

	deletedValueFlag = 0
)

var freeBatchSize int = (cacheSize * freeBatchPercent) / 100
var deletedValue = storedValue{TestValue{}, deletedValueFlag}

func init() {
	if freeBatchSize < 1 {
		freeBatchSize = 1
	}
}

type storedValue struct {
	v           TestValue
	expiredTime int
}

type Cache struct {
	storage      [cacheSize]storedValue   // Preallocated storage
	storageLocks [cacheSize]*sync.RWMutex // row level locks

	index      [indexBuckets]map[uint64]int // map[hashedKey]valueIndexInArray
	indexLocks [indexBuckets]*sync.RWMutex  // few maps for less locks

	freeIndexesLock sync.RWMutex
	freeIndexes     []int
	freeCount       *int32 // Used to store last stored index in freeIndexes (len analog)
	freeIndexCh     chan struct{}

	onClearing      *int32
	startClearingCh chan struct{}
	endClearingCh   chan struct{}

	stop chan struct{}
	sync.RWMutex
}

// TODO: добавить логер, метрику в виде определяемых интерфейсов
func NewNiceCache() *Cache {
	freeIndexes := make([]int, cacheSize)
	for i := 0; i < cacheSize; i++ {
		freeIndexes[i] = i
	}

	storageLocks := [cacheSize]*sync.RWMutex{}
	for i := 0; i < cacheSize; i++ {
		lock := sync.RWMutex{}
		storageLocks[i] = &lock
	}

	index := [indexBuckets]map[uint64]int{}
	indexLocks := [indexBuckets]*sync.RWMutex{}
	for i := 0; i < indexBuckets; i++ {
		lock := sync.RWMutex{}
		index[i] = make(map[uint64]int, cacheSize/indexBuckets)
		indexLocks[i] = &lock
	}

	n := int32(len(freeIndexes))
	freeCount := &n

	onClearing := int32(0)
	c := &Cache{
		storage:         [cacheSize]storedValue{},
		storageLocks:    storageLocks,
		index:           index,
		indexLocks:      indexLocks,
		freeIndexes:     freeIndexes,
		freeCount:       freeCount,
		onClearing:      &onClearing,
		freeIndexCh:     make(chan struct{}, freeBatchSize),
		startClearingCh: make(chan struct{}, 1),
		endClearingCh:   make(chan struct{}),
		stop:            make(chan struct{}),
	}

	go c.clearCache(c.startClearingCh)

	return c
}

func (c *Cache) Set(key []byte, value *TestValue, expireSeconds int) error {
	h := getHash(key)
	bucketIdx := getBucketIdx(h)
	indexBucketLock := c.indexLocks[bucketIdx]
	indexBucket := c.index[bucketIdx]

	indexBucketLock.RLock()
	valueIdx, ok := indexBucket[h]
	indexBucketLock.RUnlock()

	if !ok {
		valueIdx = c.popFreeIndex()

		indexBucketLock.Lock()
		indexBucket[h] = valueIdx
		indexBucketLock.Unlock()
	}

	rowLock := c.storageLocks[valueIdx]
	rowLock.Lock()
	c.storage[valueIdx].v = *value
	c.storage[valueIdx].expiredTime = int(time.Now().Unix()) + expireSeconds
	rowLock.Unlock()

	return nil
}

func (c *Cache) Get(key []byte, value *TestValue) error {
	h := getHash(key)
	bucketIdx := getBucketIdx(h)
	indexBucketLock := c.indexLocks[bucketIdx]
	indexBucket := c.index[bucketIdx]

	indexBucketLock.RLock()
	valueIdx, ok := indexBucket[h]
	indexBucketLock.RUnlock()

	if !ok {
		return NotFoundError
	}

	if value == nil {
		return NilValueError
	}

	rowLock := c.storageLocks[valueIdx]
	rowLock.RLock()
	result := c.storage[valueIdx]
	rowLock.RUnlock()

	if (result.expiredTime - int(time.Now().Unix())) <= 0 {
		c.delete(h, valueIdx)
		return NotFoundError
	}

	*value = result.v
	return nil
}

func (c *Cache) Delete(key []byte) {
	h := getHash(key)
	bucketIdx := getBucketIdx(h)
	indexBucketLock := c.indexLocks[bucketIdx]
	indexBucket := c.index[bucketIdx]

	indexBucketLock.RLock()
	valueIdx, ok := indexBucket[h]
	indexBucketLock.RUnlock()

	indexBucketLock.Lock()
	delete(indexBucket, h)
	indexBucketLock.Unlock()

	rowLock := c.storageLocks[valueIdx]
	rowLock.Lock()
	//c.storage[valueIdx] = deletedValue
	c.storage[valueIdx].expiredTime = deletedValueFlag
	rowLock.Unlock()

	if !ok {
		return
	}

	c.pushFreeIndex(valueIdx)
}

func (c *Cache) delete(h uint64, valueIdx int) {
	bucketIdx := getBucketIdx(h)
	indexBucketLock := c.indexLocks[bucketIdx]
	indexBucket := c.index[bucketIdx]

	indexBucketLock.RLock()
	valueIdx, ok := indexBucket[h]
	indexBucketLock.RUnlock()
	if !ok {
		return
	}

	indexBucketLock.Lock()
	delete(indexBucket, h)
	indexBucketLock.Unlock()

	rowLock := c.storageLocks[valueIdx]
	rowLock.Lock()
	c.storage[valueIdx] = deletedValue
	rowLock.Unlock()

	c.pushFreeIndex(valueIdx)
}

func (c *Cache) popFreeIndex() int {
	freeIdx := int(-1)
	for c.removeFreeIndex(&freeIdx) < 0 {
		endClearingCh := c.forceClearCache()
		<-endClearingCh
	}

	if freeIdx > len(c.freeIndexes)-1 || freeIdx < 0 {
		panic(freeIdx)
	}

	return c.freeIndexes[freeIdx]
}

func (c *Cache) forceClearCache() chan struct{} {
	if atomic.CompareAndSwapInt32(c.onClearing, 0, 1) {
		// Если индексы иссякли и флаг очистки не был выставлен - стартуем очистку
		c.endClearingCh = make(chan struct{})
		c.startClearingCh <- struct{}{}
	}
	return c.endClearingCh

}

func (c *Cache) clearCache(startClearingCh chan struct{}) {
	var (
		freeIdx int

		nowTime time.Time
		now     int
		cyrcle  int

		gcTicker = time.NewTicker(1 * time.Hour)

		clearChunkSize      = cacheSize / 100 // 1% over cache size
		currentChunk        int
		currentChunkIndexes [2]int
		indexInCacheArray   int

		iterateStoredValue storedValue

		freeIndexes = []int{}
		maxFreeIdx  int

		rowLock *sync.RWMutex
	)

	chunks, _ := chunks(cacheSize, clearChunkSize)

	for {
		select {
		case <-startClearingCh:
			// TODO заменить на lru?
			// Мне не нужно хранить весь список элементов с их частотой использования для lru. мне достаточно хранить НЕ БОЛЕЕ количества, которое будет очищено, то есть от freeBatchPercent до (cacheSize*maxFreeRatePercent)/100, но не менее 1. Мне не нужен большой двусвязный список, возможно удастся обойтись обычным отсортированным списком
			i := 0

			for bucketIdx, bucket := range c.index {
				indexBucketLock := c.indexLocks[bucketIdx]

				indexBucketLock.Lock()

				for h, valueIdx := range bucket {
					delete(bucket, h)

					rowLock = c.storageLocks[valueIdx]
					rowLock.Lock()
					if c.storage[valueIdx].expiredTime == deletedValueFlag {
						// trying to delete deleted element in map
						rowLock.Unlock()
						continue
					}
					c.storage[valueIdx] = deletedValue
					rowLock.Unlock()

					freeIdx = c.addFreeIndex()
					c.freeIndexes[freeIdx] = valueIdx

					i++
					if i >= freeBatchSize {
						break
					}
				}
				indexBucketLock.Unlock()
			}

			// Increase freeBatchSize progressive
			var freeBatchSizeDelta int = freeBatchSize * alpha / 100
			if freeBatchSizeDelta < 1 {
				freeBatchSizeDelta = 1
			}

			freeBatchSize += freeBatchSizeDelta
			if freeBatchSize > (cacheSize*maxFreeRatePercent)/100 {
				freeBatchSize = (cacheSize * maxFreeRatePercent) / 100
			}
			if freeBatchSize < 1 {
				freeBatchSize = 1
			}

			atomic.StoreInt32(c.onClearing, 0)
			close(c.endClearingCh)
		case nowTime = <-gcTicker.C:
			now = int(nowTime.Unix())

			currentChunk = cyrcle % len(chunks)
			currentChunkIndexes = chunks[currentChunk]

			for idx := range c.storage[currentChunkIndexes[0]:currentChunkIndexes[1]] {
				indexInCacheArray = idx + currentChunkIndexes[0]

				rowLock = c.storageLocks[indexInCacheArray]
				rowLock.RLock()
				iterateStoredValue = c.storage[indexInCacheArray]
				rowLock.RUnlock()

				if iterateStoredValue.expiredTime == deletedValueFlag {
					continue
				}

				if (iterateStoredValue.expiredTime - now) <= 0 {
					rowLock.Lock()
					c.storage[indexInCacheArray] = deletedValue
					rowLock.Unlock()

					freeIndexes = append(freeIndexes, indexInCacheArray)
				}
			}

			if len(freeIndexes) > 0 {
				maxFreeIdx = c.addNFreeIndex(len(freeIndexes))
				for _, indexInCacheArray := range freeIndexes {
					c.freeIndexes[maxFreeIdx] = indexInCacheArray
					maxFreeIdx--
				}

				// try to reuse freeIndexes slice
				if cap(freeIndexes) > 10000 {
					freeIndexes = []int{}
				}
				freeIndexes = freeIndexes[:0]
			}

			cyrcle++
		case <-c.stop:
			gcTicker.Stop()
			return
		}
	}
}

func (c *Cache) pushFreeIndex(key int) {
	freeIdx := c.addFreeIndex()
	c.freeIndexes[freeIdx] = key
}

// increase freeIndexCount and returns new last free index
func (c *Cache) addFreeIndex() int {
	return int(atomic.AddInt32(c.freeCount, int32(1))) - 1 //Idx == new freeCount - 1 == old freeCount
}

// decrease freeIndexCount and returns new last free index
func (c *Cache) removeFreeIndex(idx *int) int {
	*idx = int(atomic.AddInt32(c.freeCount, int32(-1))) //Idx == new freeCount == old freeCount - 1
	if *idx < 0 {
		atomic.AddInt32(c.freeCount, int32(1))
		return -1
	}
	return *idx
}

// increase freeIndexCount by N and returns new last free index
func (c *Cache) addNFreeIndex(n int) int {
	return int(atomic.AddInt32(c.freeCount, int32(n))) - 1 //Idx == new freeCount - 1 == old freeCount
}

func (c *Cache) Flush() {
	c.Lock()
	for bucketIdx := range c.index {
		indexBucketLock := c.indexLocks[bucketIdx]

		indexBucketLock.Lock()

		for i := 0; i < cacheSize; i++ {
			c.freeIndexes[i] = i
		}
		atomic.StoreInt32(c.freeCount, cacheSize)

		c.index[bucketIdx] = make(map[uint64]int, cacheSize)

		indexBucketLock.Unlock()
	}
	c.Unlock()
}

func (c *Cache) Len() int {
	return cacheSize - int(atomic.LoadInt32(c.freeCount))
}

func (c *Cache) Close() {
	close(c.stop)
}
