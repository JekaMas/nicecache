package nicecache

import (
	"sync"
	"sync/atomic"
	"time"
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
	storage      *[cacheSize]storedValue  // Preallocated storage
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

	stop       chan struct{}
	isStopped  *int32
	onFlushing *int32
}

func NewNiceCache() *Cache {
	return newNiceCache()
}

func newNiceCache() *Cache {
	freeIndexes := make([]int, cacheSize)
	for i := 0; i < cacheSize; i++ {
		freeIndexes[i] = i
	}

	storageLocks := [cacheSize]*sync.RWMutex{}
	for i := 0; i < cacheSize; i++ {
		storageLocks[i] = new(sync.RWMutex)
	}

	index := [indexBuckets]map[uint64]int{}
	indexLocks := [indexBuckets]*sync.RWMutex{}
	for i := 0; i < indexBuckets; i++ {
		index[i] = make(map[uint64]int, cacheSize/indexBuckets)
		indexLocks[i] = new(sync.RWMutex)
	}

	n := int32(len(freeIndexes))
	freeCount := &n

	c := &Cache{
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
	}

	go c.clearCache(c.startClearingCh)

	return c
}

func (c *Cache) Set(key []byte, value *TestValue, expireSeconds int) error {
	if c.isClosed() {
		return CloseError
	}

	h := getHash(key)
	bucketIdx := getBucketIDs(h)
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
	(*c.storage)[valueIdx].v = *value
	(*c.storage)[valueIdx].expiredTime = int(time.Now().Unix()) + expireSeconds
	rowLock.Unlock()

	return nil
}

func (c *Cache) Get(key []byte, value *TestValue) error {
	if c.isClosed() {
		return CloseError
	}

	if value == nil {
		return NilValueError
	}

	h := getHash(key)
	bucketIdx := getBucketIDs(h)
	indexBucketLock := c.indexLocks[bucketIdx]
	indexBucket := c.index[bucketIdx]

	indexBucketLock.RLock()
	valueIdx, ok := indexBucket[h]
	indexBucketLock.RUnlock()

	if !ok {
		return NotFoundError
	}

	rowLock := c.storageLocks[valueIdx]
	rowLock.RLock()
	result := (*c.storage)[valueIdx]
	rowLock.RUnlock()

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

func (c *Cache) Delete(key []byte) error {
	if c.isClosed() {
		return CloseError
	}

	h := getHash(key)
	bucketIdx := getBucketIDs(h)
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
	(*c.storage)[valueIdx] = deletedValue
	(*c.storage)[valueIdx].expiredTime = deletedValueFlag
	rowLock.Unlock()

	if !ok {
		return nil
	}

	c.pushFreeIndex(valueIdx)
	return nil
}

// deleteItem item by it bucket hash and index in bucket
func (c *Cache) deleteItem(h uint64, valueIdx int) {
	bucketIdx := getBucketIDs(h)
	indexBucketLock := c.indexLocks[bucketIdx]
	indexBucket := c.index[bucketIdx]

	/*
		indexBucketLock.RLock()
		valueIdx, ok := indexBucket[h]
		indexBucketLock.RUnlock()
		if !ok {
			return
		}
	*/

	//
	indexBucketLock.Lock()
	delete(indexBucket, h)
	indexBucketLock.Unlock()

	rowLock := c.storageLocks[valueIdx]
	rowLock.Lock()
	(*c.storage)[valueIdx] = deletedValue
	rowLock.Unlock()

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

	if freeIdx > len(c.freeIndexes)-1 || freeIdx < 0 {
		// fixme dont panic
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

// push back index to mark it as free in cache storage
func (c *Cache) pushFreeIndex(valueIdx int) {
	freeIdx := c.addFreeIndex()

	c.freeIndexes[freeIdx] = valueIdx
}

// increase freeIndexCount and returns new last free index
func (c *Cache) addFreeIndex() int {
	return int(atomic.AddInt32(c.freeCount, int32(1))) - 1 //Idx == new freeCount - 1 == old freeCount
}

// decrease freeIndexCount and returns new last free index
// todo check if i can use atomic.Cond for this
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

		rowLock *sync.RWMutex
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

			i := 0

			for bucketIdx, bucket := range c.index {
				indexBucketLock := c.indexLocks[bucketIdx]

				indexBucketLock.Lock()

				for h, valueIdx := range bucket {
					delete(bucket, h)

					rowLock = c.storageLocks[valueIdx]
					rowLock.Lock()
					if (*c.storage)[valueIdx].expiredTime == deletedValueFlag {
						// trying to deleteItem deleted element in map
						rowLock.Unlock()
						continue
					}
					(*c.storage)[valueIdx] = deletedValue
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
			// by time garbage collector
			if c.isClosed() {
				gcTicker.Stop()
				return
			}

			now = int(nowTime.Unix())

			currentChunk = circle % len(chunks)
			currentChunkIndexes = chunks[currentChunk]

			for idx := range (*c.storage)[currentChunkIndexes[0]:currentChunkIndexes[1]] {
				indexInCacheArray = idx + currentChunkIndexes[0]

				rowLock = c.storageLocks[indexInCacheArray]
				rowLock.RLock()
				iterateStoredValue = (*c.storage)[indexInCacheArray]
				rowLock.RUnlock()

				if iterateStoredValue.expiredTime == deletedValueFlag {
					continue
				}

				if (iterateStoredValue.expiredTime - now) <= 0 {
					rowLock.Lock()
					(*c.storage)[indexInCacheArray] = deletedValue
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

			circle++
		//todo сюда стоит поместить выполнение кода Flush, тогда я гарантирую, что он не будет идти одновременно с другими операциями gc
		case <-c.stop:
			gcTicker.Stop()
			return
		}
	}
}

func (c *Cache) Flush() error {
	if c.isClosed() {
		return CloseError
	}

	newCache := newNiceCache()

	c.Close()

	// todo resolve data race
	c.storage = newCache.storage
	c.storageLocks = newCache.storageLocks

	c.index = newCache.index
	c.indexLocks = newCache.indexLocks

	c.freeIndexesLock = newCache.freeIndexesLock
	c.freeIndexes = newCache.freeIndexes
	c.freeCount = newCache.freeCount
	c.freeIndexCh = newCache.freeIndexCh

	c.onClearing = newCache.onClearing
	c.startClearingCh = newCache.startClearingCh
	c.endClearingCh = newCache.endClearingCh

	c.stop = newCache.stop
	c.onFlushing = newCache.onFlushing

	// should be last
	c.isStopped = newCache.isStopped

	return nil
}

func (c *Cache) Len() int {
	if c.isClosed() {
		return 0
	}

	return cacheSize - int(atomic.LoadInt32(c.freeCount))
}

func (c *Cache) Close() {
	if c.isClosed() {
		return
	}

	close(c.stop)
	atomic.StoreInt32(c.isStopped, 1)

	/*
	go func() {
		//todo is this best solution?
		time.Sleep(5 * time.Second)

		c.storage = nil
		c.freeIndexes = nil

		for i := 0; i < cacheSize; i++ {
			c.storageLocks[i] = nil
		}

		for i := 0; i < indexBuckets; i++ {
			c.index[i] = nil
			c.indexLocks[i] = nil
		}
	}()
	*/
}

func (c *Cache) isClosed() bool {
	return atomic.LoadInt32(c.isStopped) == 1
}
