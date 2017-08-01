package internal

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	cacheSize    = 1024 * 1024 * 10
	indexBuckets = 100 // how many buckets for cache items

	// forced gc
	freeBatchPercent   = 1  // how many items gc process in one step, in percent
	alpha              = 1  // Percent increasing speed of freeBatchSize
	maxFreeRatePercent = 33 // maximal number of items for one gc step, in percent

	// normal gc
	// GC full circle takes time = gcTime*(cacheSize/gcChunkSize)
	// TODO: gcTime and gcChunkPercent could be tuned manually or automatic
	gcTime         = 1 * time.Second                  // how often gc runs
	gcChunkPercent = 1                                // percent of items to proceed in gc step
	gcChunkSize    = cacheSize * gcChunkPercent / 100 // number of items to proceed in gc step

	deletedValueFlag = 0
)

var freeBatchSize int = (cacheSize * freeBatchPercent) / 100
var deletedValue = storedValue{TestValue{}, deletedValueFlag}

// todo move into generator
func init() {
	if freeBatchSize < 1 {
		freeBatchSize = 1
	}
}

type storedValue struct {
	v           TestValue
	expiredTime int
}

//Cache typed cache
type Cache struct {
	cache *cache
}

type cache struct {
	storage      *[cacheSize]storedValue  // Preallocated storage
	storageLocks [cacheSize]sync.RWMutex // row level locks

	index      [indexBuckets]map[uint64]int // map[hashedKey]valueIndexInArray
	indexLocks [indexBuckets]sync.RWMutex  // few maps for less locks

	freeIndexesLock sync.RWMutex
	freeIndexes     []int
	freeCount       *int32 // Used to store last stored index in freeIndexes (len analog)
	freeIndexCh     chan struct{}

	onClearing      *int32
	startClearingCh chan struct{}
	endClearingCh   chan struct{}

	stop       chan struct{}
	isStopped  *int32
	onFlushing *int32
}

//NewNiceCache typed cache constructor
func NewNiceCache() *Cache {
	return newNiceCache()
}

func newNiceCache() *Cache {
	freeIndexes := make([]int, cacheSize)
	for i := 0; i < cacheSize; i++ {
		freeIndexes[i] = i
	}

	storageLocks := [cacheSize]sync.RWMutex{}
	indexLocks := [indexBuckets]sync.RWMutex{}

	index := [indexBuckets]map[uint64]int{}
	for i := 0; i < indexBuckets; i++ {
		index[i] = make(map[uint64]int, cacheSize/indexBuckets)
	}

	n := int32(len(freeIndexes))
	freeCount := &n

	c := &Cache{
		&cache{
			storage:         new([cacheSize]storedValue),
			storageLocks:    storageLocks,
			index:           index,
			indexLocks:      indexLocks,
			freeIndexes:     freeIndexes,
			freeCount:       freeCount,
			onClearing:      new(int32),
			freeIndexCh:     make(chan struct{}, freeBatchSize),
			startClearingCh: make(chan struct{}, 1),
			endClearingCh:   make(chan struct{}),
			stop:            make(chan struct{}),
			onFlushing:      new(int32),
			isStopped:       new(int32),
		},
	}

	go c.clearCache(c.cache.startClearingCh)

	return c
}

//Set value by key
func (c *Cache) Set(key []byte, value *TestValue, expireSeconds int) error {
	if c.isClosed() {
		return CloseError
	}

	h := getHash(key)
	bucketIdx := getBucketIDs(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].RLock()
	valueIdx, ok := indexBucket[h]
	c.cache.indexLocks[bucketIdx].RUnlock()

	if !ok {
		valueIdx = c.popFreeIndex()

		c.cache.indexLocks[bucketIdx].Lock()
		indexBucket[h] = valueIdx
		c.cache.indexLocks[bucketIdx].Unlock()
	}

	c.cache.storageLocks[valueIdx].Lock()
	(*c.cache.storage)[valueIdx].v = *value
	(*c.cache.storage)[valueIdx].expiredTime = int(time.Now().Unix()) + expireSeconds
	c.cache.storageLocks[valueIdx].Unlock()

	return nil
}

//Get value by key
func (c *Cache) Get(key []byte, value *TestValue) error {
	if c.isClosed() {
		return CloseError
	}

	if value == nil {
		return NilValueError
	}

	h := getHash(key)
	bucketIdx := getBucketIDs(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].RLock()
	valueIdx, ok := indexBucket[h]
	c.cache.indexLocks[bucketIdx].RUnlock()

	if !ok {
		return NotFoundError
	}

	c.cache.storageLocks[valueIdx].RLock()
	result := (*c.cache.storage)[valueIdx]
	c.cache.storageLocks[valueIdx].RUnlock()

	if result.expiredTime == deletedValueFlag {
		return NotFoundError
	}

	if (result.expiredTime - int(time.Now().Unix())) <= 0 {
		c.deleteItem(h, valueIdx)
		return NotFoundError
	}

	*value = result.v
	return nil
}

//Delete value by key
func (c *Cache) Delete(key []byte) error {
	if c.isClosed() {
		return CloseError
	}

	h := getHash(key)
	bucketIdx := getBucketIDs(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].RLock()
	valueIdx, ok := indexBucket[h]
	c.cache.indexLocks[bucketIdx].RUnlock()

	c.cache.indexLocks[bucketIdx].Lock()
	delete(indexBucket, h)
	c.cache.indexLocks[bucketIdx].Unlock()

	c.cache.storageLocks[valueIdx].Lock()
	(*c.cache.storage)[valueIdx] = deletedValue
	(*c.cache.storage)[valueIdx].expiredTime = deletedValueFlag
	c.cache.storageLocks[valueIdx].Unlock()

	if !ok {
		return nil
	}

	c.pushFreeIndex(valueIdx)
	return nil
}

// deleteItem item by it bucket hash and index in bucket
func (c *Cache) deleteItem(h uint64, valueIdx int) {
	bucketIdx := getBucketIDs(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].Lock()
	delete(indexBucket, h)
	c.cache.indexLocks[bucketIdx].Unlock()

	c.cache.storageLocks[valueIdx].Lock()
	(*c.cache.storage)[valueIdx] = deletedValue
	c.cache.storageLocks[valueIdx].Unlock()

	c.pushFreeIndex(valueIdx)
}

// get one index from storage to store new item
func (c *Cache) popFreeIndex() int {
	freeIdx := int(-1)
	for c.removeFreeIndex(&freeIdx) < 0 {
		// all cache is full
		endClearingCh := c.forceClearCache()
		<-endClearingCh
	}

	if freeIdx > len(c.cache.freeIndexes)-1 || freeIdx < 0 {
		// fixme dont panic
		panic(freeIdx)
	}

	c.cache.freeIndexesLock.RLock()
	freeIDx := c.cache.freeIndexes[freeIdx]
	c.cache.freeIndexesLock.RUnlock()
	return freeIDx
}

func (c *Cache) forceClearCache() chan struct{} {
	if atomic.CompareAndSwapInt32(c.cache.onClearing, 0, 1) {
		// Если индексы иссякли и флаг очистки не был выставлен - стартуем очистку
		c.cache.endClearingCh = make(chan struct{})
		c.cache.startClearingCh <- struct{}{}
	}
	return c.cache.endClearingCh
}

// push back index to mark it as free in cache storage
func (c *Cache) pushFreeIndex(valueIdx int) {
	freeIdx := c.addFreeIndex()

	c.cache.freeIndexesLock.Lock()
	c.cache.freeIndexes[freeIdx] = valueIdx
	c.cache.freeIndexesLock.Unlock()
}

// increase freeIndexCount and returns new last free index
func (c *Cache) addFreeIndex() int {
	return int(atomic.AddInt32(c.cache.freeCount, int32(1))) - 1 //Idx == new freeCount - 1 == old freeCount
}

// decrease freeIndexCount and returns new last free index
// todo check if i can use atomic.Cond for messaging all waiting for new free ids goroutines
func (c *Cache) removeFreeIndex(idx *int) int {
	*idx = int(atomic.AddInt32(c.cache.freeCount, int32(-1))) //Idx == new freeCount == old freeCount - 1
	if *idx < 0 {
		atomic.AddInt32(c.cache.freeCount, int32(1))
		return -1
	}
	return *idx
}

// increase freeIndexCount by N and returns new last free index
func (c *Cache) addNFreeIndex(n int) int {
	return int(atomic.AddInt32(c.cache.freeCount, int32(n))) - 1 //Idx == new freeCount - 1 == old freeCount
}

func (c *Cache) clearCache(startClearingCh chan struct{}) {
	var (
		freeIdx int

		nowTime time.Time
		now     int
		circle  int

		gcTicker = time.NewTicker(gcTime)

		currentChunk        int
		currentChunkIndexes [2]int
		indexInCacheArray   int

		iterateStoredValue storedValue

		freeIndexes = []int{}
		maxFreeIdx  int
	)

	// even for strange gcChunkSize chunks func guarantees that all indexes will present in result chunks
	chunks, _ := chunks(cacheSize, gcChunkSize)

	for {
		select {
		case <-startClearingCh:
			// forced gc
			if c.isClosed() {
				gcTicker.Stop()
				return
			}

			//todo add more cleaning cache strategies
			c.randomKeyCleanCacheStrategy(freeIdx)

			// Increase freeBatchSize progressive
			c.setNewFreeBatchSize()

			atomic.StoreInt32(c.cache.onClearing, 0)
			close(c.cache.endClearingCh)
		case nowTime = <-gcTicker.C:
			// by time garbage collector
			if c.isClosed() {
				gcTicker.Stop()
				return
			}

			now = int(nowTime.Unix())

			currentChunk = circle % len(chunks)
			currentChunkIndexes = chunks[currentChunk]

			for idx := range (*c.cache.storage)[currentChunkIndexes[0]:currentChunkIndexes[1]] {
				indexInCacheArray = idx + currentChunkIndexes[0]

				c.cache.storageLocks[indexInCacheArray].RLock()
				iterateStoredValue = (*c.cache.storage)[indexInCacheArray]
				c.cache.storageLocks[indexInCacheArray].RUnlock()

				if iterateStoredValue.expiredTime == deletedValueFlag {
					continue
				}

				if (iterateStoredValue.expiredTime - now) <= 0 {
					c.cache.storageLocks[indexInCacheArray].Lock()
					(*c.cache.storage)[indexInCacheArray] = deletedValue
					c.cache.storageLocks[indexInCacheArray].Unlock()

					freeIndexes = append(freeIndexes, indexInCacheArray)
				}
			}

			if len(freeIndexes) > 0 {
				maxFreeIdx = c.addNFreeIndex(len(freeIndexes))
				c.cache.freeIndexesLock.Lock()
				for _, indexInCacheArray := range freeIndexes {
					c.cache.freeIndexes[maxFreeIdx] = indexInCacheArray
					maxFreeIdx--
				}
				c.cache.freeIndexesLock.Unlock()

				// try to reuse freeIndexes slice
				if cap(freeIndexes) > 10000 {
					freeIndexes = []int{}
				}
				freeIndexes = freeIndexes[:0]
			}

			circle++
		case <-c.cache.stop:
			gcTicker.Stop()
			return
		}
	}
}

func (*Cache) setNewFreeBatchSize() {
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
}

func (c *Cache) randomKeyCleanCacheStrategy(freeIdx int) {
	i := 0

	for bucketIdx, bucket := range c.cache.index {
		c.cache.indexLocks[bucketIdx].Lock()

		for h, valueIdx := range bucket {
			delete(bucket, h)

			c.cache.storageLocks[valueIdx].Lock()
			if (*c.cache.storage)[valueIdx].expiredTime == deletedValueFlag {
				// trying to deleteItem deleted element in map
				c.cache.storageLocks[valueIdx].Unlock()
				continue
			}
			(*c.cache.storage)[valueIdx] = deletedValue
			c.cache.storageLocks[valueIdx].Unlock()

			freeIdx = c.addFreeIndex()

			c.cache.freeIndexesLock.Lock()
			c.cache.freeIndexes[freeIdx] = valueIdx
			c.cache.freeIndexesLock.Unlock()

			i++
			if i >= freeBatchSize {
				break
			}
		}
		c.cache.indexLocks[bucketIdx].Unlock()
	}
}

//Flush cache. Can take a time
func (c *Cache) Flush() error {
	if c.isClosed() {
		return CloseError
	}

	c.Close()
	//todo can i do better?
	time.Sleep(1*time.Second)

	newCache := newNiceCache()

	// atomic store new cache
	oldPtr := (*unsafe.Pointer)(unsafe.Pointer(&c.cache))
	newUnsafe := unsafe.Pointer(newCache.cache)
	atomic.StorePointer(oldPtr, newUnsafe)

	return nil
}

//Len get stored elements count
func (c *Cache) Len() int {
	if c.isClosed() {
		return 0
	}

	return cacheSize - int(atomic.LoadInt32(c.cache.freeCount))
}

//Close cache
func (c *Cache) Close() {
	if c.isClosed() {
		return
	}

	close(c.cache.stop)
	atomic.StoreInt32(c.cache.isStopped, 1)
}

func (c *Cache) isClosed() bool {
	return atomic.LoadInt32(c.cache.isStopped) == 1
}
