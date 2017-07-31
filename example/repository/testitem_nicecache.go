/*
* CODE GENERATED AUTOMATICALLY WITH github.com/jekamas/nicecache
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

package repository

import (
    "sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/dgryski/go-farm"
	. "github.com/JekaMas/nicecache/example"
)

/* ___________ bucket_hash ___________ */

func getBucketIDsTestItem(h uint64) int {
	return int(h % 100)
}

/* ___________ chunks ___________ */

// chunks returns equals chunks
func chunksTestItem(sliceLen, chunkSize int) ([][2]int, error) {
	ranges := [][2]int{}

	if sliceLen < 0 {
		return nil, chunksNegativeSliceSizeTestItem
	}

	if chunkSize <= 0 {
		return nil, chunksNegativeSizeTestItem
	}

	var prevI int
	for i := 0; i < sliceLen; i++ {
		if sliceLen != 1 && i != sliceLen-1 && ((i+1)%chunkSize != 0 || i == 0) {
			continue
		}

		ranges = append(ranges, [2]int{prevI, i + 1})

		prevI = i + 1
	}

	return ranges, nil
}

/* ___________ errors ___________ */

type cacheErrorTestItem string

func (e cacheErrorTestItem) Error() string { return string(e) }

const (
	NotFoundErrorTestItem = cacheErrorTestItem("key not found")
	NilValueErrorTestItem = cacheErrorTestItem("value pointer shouldnt be nil")
	CloseErrorTestItem    = cacheErrorTestItem("cache has been closed")

	chunksNegativeSliceSizeTestItem = cacheErrorTestItem("sliceLen should be non-negative")
	chunksNegativeSizeTestItem = cacheErrorTestItem("chunkSize should be positive")
)

/* ___________ key_hash ___________ */

func getHashTestItem(key []byte) uint64 {
	return farm.Hash64(key)
}

/* ___________ nicecache ___________ */

const (
	cacheSizeTestItem    = 10000000
	indexBucketsTestItem = 100 // how many buckets for cache items

	// forced gc
	freeBatchPercentTestItem   = 1  // how many items gc process in one step, in percent
	alphaTestItem              = 1  // Percent increasing speed of freeBatchSize
	maxFreeRatePercentTestItem = 33 // maximal number of items for one gc step, in percent

	// normal gc
	// GC full circle takes time = gcTimeTestItem*(cacheSizeTestItem/gcChunkSizeTestItem)
	gcTimeTestItem         = 1 * time.Second                  // how often gc runs
	gcChunkPercentTestItem = 1                                // percent of items to proceed in gc step
	gcChunkSizeTestItem    = cacheSizeTestItem * gcChunkPercentTestItem / 100 // number of items to proceed in gc step

	deletedValueFlagTestItem = 0
)

var freeBatchSizeTestItem int = (cacheSizeTestItem * freeBatchPercentTestItem) / 100
var deletedValueTestItem = storedValueTestItem{ TestItem{}, deletedValueFlagTestItem }

func init() {
	if freeBatchSizeTestItem < 1 {
		freeBatchSizeTestItem = 1
	}
}

type storedValueTestItem struct {
	v           TestItem
	expiredTime int
}

type CacheTestItem struct {
	cache *innerCacheTestItem
}

type innerCacheTestItem struct {
	storage      *[cacheSizeTestItem]storedValueTestItem  // Preallocated storage
	storageLocks [cacheSizeTestItem]*sync.RWMutex // row level locks

	index      [indexBucketsTestItem]map[uint64]int // map[hashedKey]valueIndexInArray
	indexLocks [indexBucketsTestItem]*sync.RWMutex  // few maps for less locks

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

func NewNiceCacheTestItem() *CacheTestItem {
	return newNiceCacheTestItem()
}

func newNiceCacheTestItem() *CacheTestItem {
	freeIndexes := make([]int, cacheSizeTestItem)
	for i := 0; i < cacheSizeTestItem; i++ {
		freeIndexes[i] = i
	}

	storageLocks := [cacheSizeTestItem]*sync.RWMutex{}
	for i := 0; i < cacheSizeTestItem; i++ {
		storageLocks[i] = new(sync.RWMutex)
	}

	index := [indexBucketsTestItem]map[uint64]int{}
	indexLocks := [indexBucketsTestItem]*sync.RWMutex{}
	for i := 0; i < indexBucketsTestItem; i++ {
		index[i] = make(map[uint64]int, cacheSizeTestItem/indexBucketsTestItem)
		indexLocks[i] = new(sync.RWMutex)
	}

	n := int32(len(freeIndexes))
	freeCount := &n

	c := &CacheTestItem{
		&innerCacheTestItem{
			storage:         new([cacheSizeTestItem]storedValueTestItem),
			storageLocks:    storageLocks,
			index:           index,
			indexLocks:      indexLocks,
			freeIndexes:     freeIndexes,
			freeCount:       freeCount,
			onClearing:      new(int32),
			freeIndexCh:     make(chan struct{}, freeBatchSizeTestItem),
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

func (c *CacheTestItem) Set(key []byte, value *TestItem, expireSeconds int) error {
	if c.isClosed() {
		return CloseErrorTestItem
	}

	h := getHashTestItem(key)
	bucketIdx := getBucketIDsTestItem(h)
	indexBucketLock := c.cache.indexLocks[bucketIdx]
	indexBucket := c.cache.index[bucketIdx]

	indexBucketLock.RLock()
	valueIdx, ok := indexBucket[h]
	indexBucketLock.RUnlock()

	if !ok {
		valueIdx = c.popFreeIndex()

		indexBucketLock.Lock()
		indexBucket[h] = valueIdx
		indexBucketLock.Unlock()
	}

	rowLock := c.cache.storageLocks[valueIdx]
	rowLock.Lock()
	(*c.cache.storage)[valueIdx].v = *value
	(*c.cache.storage)[valueIdx].expiredTime = int(time.Now().Unix()) + expireSeconds
	rowLock.Unlock()

	return nil
}

func (c *CacheTestItem) Get(key []byte, value *TestItem) error {
	if c.isClosed() {
		return CloseErrorTestItem
	}

	if value == nil {
		return NilValueErrorTestItem
	}

	h := getHashTestItem(key)
	bucketIdx := getBucketIDsTestItem(h)
	indexBucketLock := c.cache.indexLocks[bucketIdx]
	indexBucket := c.cache.index[bucketIdx]

	indexBucketLock.RLock()
	valueIdx, ok := indexBucket[h]
	indexBucketLock.RUnlock()

	if !ok {
		return NotFoundErrorTestItem
	}

	rowLock := c.cache.storageLocks[valueIdx]
	rowLock.RLock()
	result := (*c.cache.storage)[valueIdx]
	rowLock.RUnlock()

	if result.expiredTime == deletedValueFlagTestItem {
		return NotFoundErrorTestItem
	}

	if (result.expiredTime - int(time.Now().Unix())) <= 0 {
		c.deleteItem(h, valueIdx)
		return NotFoundErrorTestItem
	}

	*value = result.v
	return nil
}

func (c *CacheTestItem) Delete(key []byte) error {
	if c.isClosed() {
		return CloseErrorTestItem
	}

	h := getHashTestItem(key)
	bucketIdx := getBucketIDsTestItem(h)
	indexBucketLock := c.cache.indexLocks[bucketIdx]
	indexBucket := c.cache.index[bucketIdx]

	indexBucketLock.RLock()
	valueIdx, ok := indexBucket[h]
	indexBucketLock.RUnlock()

	indexBucketLock.Lock()
	delete(indexBucket, h)
	indexBucketLock.Unlock()

	rowLock := c.cache.storageLocks[valueIdx]
	rowLock.Lock()
	(*c.cache.storage)[valueIdx] = deletedValueTestItem
	(*c.cache.storage)[valueIdx].expiredTime = deletedValueFlagTestItem
	rowLock.Unlock()

	if !ok {
		return nil
	}

	c.pushFreeIndex(valueIdx)
	return nil
}

// deleteItem item by it bucket hash and index in bucket
func (c *CacheTestItem) deleteItem(h uint64, valueIdx int) {
	bucketIdx := getBucketIDsTestItem(h)
	indexBucketLock := c.cache.indexLocks[bucketIdx]
	indexBucket := c.cache.index[bucketIdx]

	indexBucketLock.Lock()
	delete(indexBucket, h)
	indexBucketLock.Unlock()

	rowLock := c.cache.storageLocks[valueIdx]
	rowLock.Lock()
	(*c.cache.storage)[valueIdx] = deletedValueTestItem
	rowLock.Unlock()

	c.pushFreeIndex(valueIdx)
}

// get one index from storage to store new item
func (c *CacheTestItem) popFreeIndex() int {
	freeIdx := int(-1)
	for c.removeFreeIndex(&freeIdx) < 0 {
		// all cache is full
		endClearingCh := c.forceClearCache()
		<-endClearingCh
	}

	return c.cache.freeIndexes[freeIdx]
}

func (c *CacheTestItem) forceClearCache() chan struct{} {
	if atomic.CompareAndSwapInt32(c.cache.onClearing, 0, 1) {
		c.cache.endClearingCh = make(chan struct{})
		c.cache.startClearingCh <- struct{}{}
	}
	return c.cache.endClearingCh
}

// push back index to mark it as free in cache storage
func (c *CacheTestItem) pushFreeIndex(valueIdx int) {
	freeIdx := c.addFreeIndex()

	c.cache.freeIndexes[freeIdx] = valueIdx
}

// increase freeIndexCount and returns new last free index
func (c *CacheTestItem) addFreeIndex() int {
	return int(atomic.AddInt32(c.cache.freeCount, int32(1))) - 1 //Idx == new freeCount - 1 == old freeCount
}

// decrease freeIndexCount and returns new last free index
func (c *CacheTestItem) removeFreeIndex(idx *int) int {
	*idx = int(atomic.AddInt32(c.cache.freeCount, int32(-1))) //Idx == new freeCount == old freeCount - 1
	if *idx < 0 {
		atomic.AddInt32(c.cache.freeCount, int32(1))
		return -1
	}
	return *idx
}

// increase freeIndexCount by N and returns new last free index
func (c *CacheTestItem) addNFreeIndex(n int) int {
	return int(atomic.AddInt32(c.cache.freeCount, int32(n))) - 1 //Idx == new freeCount - 1 == old freeCount
}

func (c *CacheTestItem) clearCache(startClearingCh chan struct{}) {
	var (
		freeIdx int

		nowTime time.Time
		now     int
		circle  int

		gcTicker = time.NewTicker(gcTimeTestItem)

		currentChunk        int
		currentChunkIndexes [2]int
		indexInCacheArray   int

		iterateStoredValue storedValueTestItem

		freeIndexes = []int{}
		maxFreeIdx  int

		rowLock *sync.RWMutex
	)

	// even for strange gcChunkSizeTestItem chunks func guarantees that all indexes will present in result chunks
	chunks, _ := chunksTestItem(cacheSizeTestItem, gcChunkSizeTestItem)

	for {
		select {
		case <-startClearingCh:
			// forced gc
			if c.isClosed() {
				gcTicker.Stop()
				return
			}

			i := 0

			for bucketIdx, bucket := range c.cache.index {
				indexBucketLock := c.cache.indexLocks[bucketIdx]

				indexBucketLock.Lock()

				for h, valueIdx := range bucket {
					delete(bucket, h)

					rowLock = c.cache.storageLocks[valueIdx]
					rowLock.Lock()
					if (*c.cache.storage)[valueIdx].expiredTime == deletedValueFlagTestItem {
						// trying to deleteItem deleted element in map
						rowLock.Unlock()
						continue
					}
					(*c.cache.storage)[valueIdx] = deletedValueTestItem
					rowLock.Unlock()

					freeIdx = c.addFreeIndex()
					c.cache.freeIndexes[freeIdx] = valueIdx

					i++
					if i >= freeBatchSizeTestItem {
						break
					}
				}
				indexBucketLock.Unlock()
			}

			// Increase freeBatchSizeTestItem progressive
			var freeBatchSizeDelta int = freeBatchSizeTestItem * alphaTestItem / 100
			if freeBatchSizeDelta < 1 {
				freeBatchSizeDelta = 1
			}

			freeBatchSizeTestItem += freeBatchSizeDelta
			if freeBatchSizeTestItem > (cacheSizeTestItem*maxFreeRatePercentTestItem)/100 {
				freeBatchSizeTestItem = (cacheSizeTestItem * maxFreeRatePercentTestItem) / 100
			}
			if freeBatchSizeTestItem < 1 {
				freeBatchSizeTestItem = 1
			}

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

				rowLock = c.cache.storageLocks[indexInCacheArray]
				rowLock.RLock()
				iterateStoredValue = (*c.cache.storage)[indexInCacheArray]
				rowLock.RUnlock()

				if iterateStoredValue.expiredTime == deletedValueFlagTestItem {
					continue
				}

				if (iterateStoredValue.expiredTime - now) <= 0 {
					rowLock.Lock()
					(*c.cache.storage)[indexInCacheArray] = deletedValueTestItem
					rowLock.Unlock()

					freeIndexes = append(freeIndexes, indexInCacheArray)
				}
			}

			if len(freeIndexes) > 0 {
				maxFreeIdx = c.addNFreeIndex(len(freeIndexes))
				for _, indexInCacheArray := range freeIndexes {
					c.cache.freeIndexes[maxFreeIdx] = indexInCacheArray
					maxFreeIdx--
				}

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

func (c *CacheTestItem) Flush() error {
	if c.isClosed() {
		return CloseErrorTestItem
	}

	newCache := newNiceCacheTestItem()

	c.Close()

	// atomic store new cache
	oldPtr := (*unsafe.Pointer)(unsafe.Pointer(&c.cache))
	newUnsafe := unsafe.Pointer(newCache.cache)
	atomic.StorePointer(oldPtr, newUnsafe)

	return nil
}

func (c *CacheTestItem) Len() int {
	if c.isClosed() {
		return 0
	}

	return cacheSizeTestItem - int(atomic.LoadInt32(c.cache.freeCount))
}

func (c *CacheTestItem) Close() {
	if c.isClosed() {
		return
	}

	close(c.cache.stop)
	atomic.StoreInt32(c.cache.isStopped, 1)
}

func (c *CacheTestItem) isClosed() bool {
	return atomic.LoadInt32(c.cache.isStopped) == 1
}