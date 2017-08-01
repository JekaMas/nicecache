/*
* CODE GENERATED AUTOMATICALLY WITH github.com/jekamas/nicecache
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

package example

import (
    "sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/dgryski/go-farm"
)

/* ___________ bucket_hash ___________ */

func getBucketIDsTestValue(h uint64) int {
	return int(h % 100)
}

/* ___________ chunks ___________ */

// chunks returns equals chunks
func chunksTestValue(sliceLen, chunkSize int) ([][2]int, error) {
	ranges := [][2]int{}

	if sliceLen < 0 {
		return nil, chunksNegativeSliceSizeTestValue
	}

	if chunkSize <= 0 {
		return nil, chunksNegativeSizeTestValue
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

type cacheErrorTestValue string

func (e cacheErrorTestValue) Error() string { return string(e) }

const (
	//NotFoundErrorTestValue not found error
	NotFoundErrorTestValue = cacheErrorTestValue("key not found")
	//NilValueErrorTestValue nil pointer given
	NilValueErrorTestValue = cacheErrorTestValue("value pointer shouldnt be nil")
	//CloseErrorTestValue cache is closed
	CloseErrorTestValue    = cacheErrorTestValue("cache has been closed")

	chunksNegativeSliceSizeTestValue = cacheErrorTestValue("sliceLen should be non-negative")
	chunksNegativeSizeTestValue = cacheErrorTestValue("chunkSize should be positive")
)

/* ___________ key_hash ___________ */

func getHashTestValue(key []byte) uint64 {
	return farm.Hash64(key)
}

/* ___________ nicecache ___________ */

const (
	cacheSizeTestValue    = 10000000
	indexBucketsTestValue = 100 // how many buckets for cache items

	// forced gc
	freeBatchPercentTestValue   = 1  // how many items gc process in one step, in percent
	alphaTestValue              = 1  // Percent increasing speed of freeBatchSize
	maxFreeRatePercentTestValue = 33 // maximal number of items for one gc step, in percent

	// normal gc
	// GC full circle takes time = gcTimeTestValue*(cacheSizeTestValue/gcChunkSizeTestValue)
	gcTimeTestValue         = 1 * time.Second                  // how often gc runs
	gcChunkPercentTestValue = 1                                // percent of items to proceed in gc step
	gcChunkSizeTestValue    = cacheSizeTestValue * gcChunkPercentTestValue / 100 // number of items to proceed in gc step

	deletedValueFlagTestValue = 0
)

var freeBatchSizeTestValue int = (cacheSizeTestValue * freeBatchPercentTestValue) / 100
var deletedValueTestValue = storedValueTestValue{ TestValue{}, deletedValueFlagTestValue }

func init() {
	if freeBatchSizeTestValue < 1 {
		freeBatchSizeTestValue = 1
	}
}

type storedValueTestValue struct {
	v           TestValue
	expiredTime int
}

//CacheTestValue typed cache
type CacheTestValue struct {
	cache *innerCacheTestValue
}

type innerCacheTestValue struct {
	storage      *[cacheSizeTestValue]storedValueTestValue  // Preallocated storage
	storageLocks [cacheSizeTestValue]sync.RWMutex // row level locks

	index      [indexBucketsTestValue]map[uint64]int // map[hashedKey]valueIndexInArray
	indexLocks [indexBucketsTestValue]sync.RWMutex  // few maps for less locks

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

//NewNiceCacheTestValue typed cache constructor
func NewNiceCacheTestValue() *CacheTestValue {
	return newNiceCacheTestValue()
}

func newNiceCacheTestValue() *CacheTestValue {
	freeIndexes := make([]int, cacheSizeTestValue)
	for i := 0; i < cacheSizeTestValue; i++ {
		freeIndexes[i] = i
	}

	index := [indexBucketsTestValue]map[uint64]int{}
	for i := 0; i < indexBucketsTestValue; i++ {
		index[i] = make(map[uint64]int, cacheSizeTestValue/indexBucketsTestValue)
	}

	n := int32(len(freeIndexes))
	freeCount := &n

	c := &CacheTestValue{
		&innerCacheTestValue{
			storage:         new([cacheSizeTestValue]storedValueTestValue),
			index:           index,
			freeIndexes:     freeIndexes,
			freeCount:       freeCount,
			onClearing:      new(int32),
			freeIndexCh:     make(chan struct{}, freeBatchSizeTestValue),
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
func (c *CacheTestValue) Set(key []byte, value *TestValue, expireSeconds int) error {
	if c.isClosed() {
		return CloseErrorTestValue
	}

	h := getHashTestValue(key)
	bucketIdx := getBucketIDsTestValue(h)
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
func (c *CacheTestValue) Get(key []byte, value *TestValue) error {
	if c.isClosed() {
		return CloseErrorTestValue
	}

	if value == nil {
		return NilValueErrorTestValue
	}

	h := getHashTestValue(key)
	bucketIdx := getBucketIDsTestValue(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].RLock()
	valueIdx, ok := indexBucket[h]
	c.cache.indexLocks[bucketIdx].RUnlock()

	if !ok {
		return NotFoundErrorTestValue
	}

	c.cache.storageLocks[valueIdx].RLock()
	result := (*c.cache.storage)[valueIdx]
	c.cache.storageLocks[valueIdx].RUnlock()

	if result.expiredTime == deletedValueFlagTestValue {
		return NotFoundErrorTestValue
	}

	if (result.expiredTime - int(time.Now().Unix())) <= 0 {
		c.deleteItem(h, valueIdx)
		return NotFoundErrorTestValue
	}

	*value = result.v
	return nil
}

//Delete value by key
func (c *CacheTestValue) Delete(key []byte) error {
	if c.isClosed() {
		return CloseErrorTestValue
	}

	h := getHashTestValue(key)
	bucketIdx := getBucketIDsTestValue(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].RLock()
	valueIdx, ok := indexBucket[h]
	c.cache.indexLocks[bucketIdx].RUnlock()

	c.cache.indexLocks[bucketIdx].Lock()
	delete(indexBucket, h)
	c.cache.indexLocks[bucketIdx].Unlock()

	c.cache.storageLocks[valueIdx].Lock()
	(*c.cache.storage)[valueIdx] = deletedValueTestValue
	(*c.cache.storage)[valueIdx].expiredTime = deletedValueFlagTestValue
	c.cache.storageLocks[valueIdx].Unlock()

	if !ok {
		return nil
	}

	c.pushFreeIndex(valueIdx)
	return nil
}

// deleteItem item by it bucket hash and index in bucket
func (c *CacheTestValue) deleteItem(h uint64, valueIdx int) {
	bucketIdx := getBucketIDsTestValue(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].Lock()
	delete(indexBucket, h)
	c.cache.indexLocks[bucketIdx].Unlock()

	c.cache.storageLocks[valueIdx].Lock()
	(*c.cache.storage)[valueIdx] = deletedValueTestValue
	c.cache.storageLocks[valueIdx].Unlock()

	c.pushFreeIndex(valueIdx)
}

// get one index from storage to store new item
func (c *CacheTestValue) popFreeIndex() int {
	freeIdx := int(-1)
	for c.removeFreeIndex(&freeIdx) < 0 {
		// all cache is full
		endClearingCh := c.forceClearCache()
		<-endClearingCh
	}

	c.cache.freeIndexesLock.RLock()
	freeIDx := c.cache.freeIndexes[freeIdx]
	c.cache.freeIndexesLock.RUnlock()
	return freeIDx
}

func (c *CacheTestValue) forceClearCache() chan struct{} {
	if atomic.CompareAndSwapInt32(c.cache.onClearing, 0, 1) {
		c.cache.endClearingCh = make(chan struct{})
		c.cache.startClearingCh <- struct{}{}
	}
	return c.cache.endClearingCh
}

// push back index to mark it as free in cache storage
func (c *CacheTestValue) pushFreeIndex(valueIdx int) {
	freeIdx := c.addFreeIndex()

	c.cache.freeIndexesLock.Lock()
	c.cache.freeIndexes[freeIdx] = valueIdx
	c.cache.freeIndexesLock.Unlock()
}

// increase freeIndexCount and returns new last free index
func (c *CacheTestValue) addFreeIndex() int {
	return int(atomic.AddInt32(c.cache.freeCount, int32(1))) - 1 //Idx == new freeCount - 1 == old freeCount
}

// decrease freeIndexCount and returns new last free index
func (c *CacheTestValue) removeFreeIndex(idx *int) int {
	*idx = int(atomic.AddInt32(c.cache.freeCount, int32(-1))) //Idx == new freeCount == old freeCount - 1
	if *idx < 0 {
		atomic.AddInt32(c.cache.freeCount, int32(1))
		return -1
	}
	return *idx
}

// increase freeIndexCount by N and returns new last free index
func (c *CacheTestValue) addNFreeIndex(n int) int {
	return int(atomic.AddInt32(c.cache.freeCount, int32(n))) - 1 //Idx == new freeCount - 1 == old freeCount
}

func (c *CacheTestValue) clearCache(startClearingCh chan struct{}) {
	var (
		freeIdx int

		nowTime time.Time
		now     int
		circle  int

		gcTicker = time.NewTicker(gcTimeTestValue)

		currentChunk        int
		currentChunkIndexes [2]int
		indexInCacheArray   int

		iterateStoredValue storedValueTestValue

		freeIndexes = []int{}
		maxFreeIdx  int
	)

	// even for strange gcChunkSizeTestValue chunks func guarantees that all indexes will present in result chunks
	chunks, _ := chunksTestValue(cacheSizeTestValue, gcChunkSizeTestValue)

	for {
		select {
		case <-startClearingCh:
			// forced gc
			if c.isClosed() {
				gcTicker.Stop()
				return
			}

			c.randomKeyCleanCacheStrategy(freeIdx)

			// Increase freeBatchSizeTestValue progressive
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

				if iterateStoredValue.expiredTime == deletedValueFlagTestValue {
					continue
				}

				if (iterateStoredValue.expiredTime - now) <= 0 {
					c.cache.storageLocks[indexInCacheArray].Lock()
					(*c.cache.storage)[indexInCacheArray] = deletedValueTestValue
					c.cache.storageLocks[indexInCacheArray].Unlock()

					freeIndexes = append(freeIndexes, indexInCacheArray)
				}
			}

			if len(freeIndexes) > 0 {
				maxFreeIdx = c.addNFreeIndex(len(freeIndexes))
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

func (*CacheTestValue) setNewFreeBatchSize() {
	var freeBatchSizeDelta int = freeBatchSizeTestValue * alphaTestValue / 100
	if freeBatchSizeDelta < 1 {
		freeBatchSizeDelta = 1
	}

	freeBatchSizeTestValue += freeBatchSizeDelta
	if freeBatchSizeTestValue > (cacheSizeTestValue*maxFreeRatePercentTestValue)/100 {
		freeBatchSizeTestValue = (cacheSizeTestValue * maxFreeRatePercentTestValue) / 100
	}

	if freeBatchSizeTestValue < 1 {
		freeBatchSizeTestValue = 1
	}
}

func (c *CacheTestValue) randomKeyCleanCacheStrategy(freeIdx int) {
	i := 0

	for bucketIdx, bucket := range c.cache.index {
		c.cache.indexLocks[bucketIdx].Lock()

		for h, valueIdx := range bucket {
			delete(bucket, h)

			c.cache.storageLocks[valueIdx].Lock()
			if (*c.cache.storage)[valueIdx].expiredTime == deletedValueFlagTestValue {
				// trying to deleteItem deleted element in map
				c.cache.storageLocks[valueIdx].Unlock()
				continue
			}
			(*c.cache.storage)[valueIdx] = deletedValueTestValue
			c.cache.storageLocks[valueIdx].Unlock()

			freeIdx = c.addFreeIndex()

			c.cache.freeIndexesLock.Lock()
			c.cache.freeIndexes[freeIdx] = valueIdx
			c.cache.freeIndexesLock.Unlock()

			i++
			if i >= freeBatchSizeTestValue {
				break
			}
		}
		c.cache.indexLocks[bucketIdx].Unlock()
	}
}

//Flush cache. Can take a time
func (c *CacheTestValue) Flush() error {
	if c.isClosed() {
		return CloseErrorTestValue
	}

	c.Close()

	newCache := newNiceCacheTestValue()

	// atomic store new cache
	oldPtr := (*unsafe.Pointer)(unsafe.Pointer(&c.cache))
	newUnsafe := unsafe.Pointer(newCache.cache)
	atomic.StorePointer(oldPtr, newUnsafe)

	return nil
}

//Len get stored elements count
func (c *CacheTestValue) Len() int {
	if c.isClosed() {
		return 0
	}

	return cacheSizeTestValue - int(atomic.LoadInt32(c.cache.freeCount))
}

//Close cache
func (c *CacheTestValue) Close() {
	if c.isClosed() {
		return
	}

	close(c.cache.stop)
	atomic.StoreInt32(c.cache.isStopped, 1)
}

func (c *CacheTestValue) isClosed() bool {
	return atomic.LoadInt32(c.cache.isStopped) == 1
}