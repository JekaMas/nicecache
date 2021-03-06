package template

// Code contains full code template
var Code = `/*
* CODE GENERATED AUTOMATICALLY WITH github.com/jekamas/nicecache
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

package {{ .PackageName }}

import (
    "sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/dgryski/go-farm"{{ .StoredTypePackage }}
)

/* ___________ bucket_hash ___________ */

func getBucketIDs{{ .StoredTypeSuffix }}(h uint64) int {
	return int(h % {{ .IndexBuckets }})
}

/* ___________ chunks ___________ */

// chunks returns equals chunks
func chunks{{ .StoredTypeSuffix }}(sliceLen, chunkSize int) ([][2]int, error) {
	ranges := [][2]int{}

	if sliceLen < 0 {
		return nil, chunksNegativeSliceSize{{ .StoredTypeSuffix }}
	}

	if chunkSize <= 0 {
		return nil, chunksNegativeSize{{ .StoredTypeSuffix }}
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

type cacheError{{ .StoredTypeSuffix }} string

func (e cacheError{{ .StoredTypeSuffix }}) Error() string { return string(e) }

const (
	//NotFoundError{{ .StoredTypeSuffix }} not found error
	NotFoundError{{ .StoredTypeSuffix }} = cacheError{{ .StoredTypeSuffix }}("key not found")
	//NilValueError{{ .StoredTypeSuffix }} nil pointer given
	NilValueError{{ .StoredTypeSuffix }} = cacheError{{ .StoredTypeSuffix }}("value pointer shouldnt be nil")
	//CloseError{{ .StoredTypeSuffix }} cache is closed
	CloseError{{ .StoredTypeSuffix }}    = cacheError{{ .StoredTypeSuffix }}("cache has been closed")

	chunksNegativeSliceSize{{ .StoredTypeSuffix }} = cacheError{{ .StoredTypeSuffix }}("sliceLen should be non-negative")
	chunksNegativeSize{{ .StoredTypeSuffix }} = cacheError{{ .StoredTypeSuffix }}("chunkSize should be positive")
)

/* ___________ key_hash ___________ */

func getHash{{ .StoredTypeSuffix }}(key []byte) uint64 {
	return farm.Hash64(key)
}

/* ___________ nicecache ___________ */

const (
	{{ .CacheSizeConstName }}    = {{ .CacheSize }}
	{{ .IndexBucketsName }} = 100 // how many buckets for cache items

	// forced gc
	{{ .FreeBatchPercentName }}   = 1  // how many items gc process in one step, in percent
	{{ .AlphaName }}              = 1  // Percent increasing speed of freeBatchSize
	{{ .MaxFreeRatePercentName }} = 33 // maximal number of items for one gc step, in percent

	// normal gc
	// GC full circle takes time = {{ .GcTimeName }}*({{ .CacheSizeConstName }}/{{ .GcChunkSizeName }})
	{{ .GcTimeName }}         = 1 * time.Second                  // how often gc runs
	{{ .GcChunkPercentName }} = 1                                // percent of items to proceed in gc step
	{{ .GcChunkSizeName }}    = {{ .CacheSizeConstName }} * {{ .GcChunkPercentName }} / 100 // number of items to proceed in gc step

	{{ .DeletedValueFlagName }} = 0
)

var {{ .FreeBatchSizeName }} int = ({{ .CacheSizeConstName }} * {{ .FreeBatchPercentName }}) / 100
var {{ .DeletedValueName }} = {{ .StoredValueType }}{ {{ .StoredType }}{}, {{ .DeletedValueFlagName }} }

func init() {
	if {{ .FreeBatchSizeName }} < 1 {
		{{ .FreeBatchSizeName }} = 1
	}
}

type {{ .StoredValueType }} struct {
	v           {{ .StoredType }}
	expiredTime int
}

//{{ .CacheType }} typed cache
type {{ .CacheType }} struct {
	cache *{{ .InnerCacheType }}
}

type {{ .InnerCacheType }} struct {
	storage      *[{{ .CacheSizeConstName }}]{{ .StoredValueType }}  // Preallocated storage
	storageLocks [{{ .CacheSizeConstName }}]sync.RWMutex // row level locks

	index      [{{ .IndexBucketsName }}]map[uint64]int // map[hashedKey]valueIndexInArray
	indexLocks [{{ .IndexBucketsName }}]sync.RWMutex  // few maps for less locks

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

//NewNice{{ .CacheType }} typed cache constructor
func NewNice{{ .CacheType }}() *{{ .CacheType }} {
	return newNice{{ .CacheType }}()
}

func newNice{{ .CacheType }}() *{{ .CacheType }} {
	freeIndexes := make([]int, {{ .CacheSizeConstName }})
	for i := 0; i < {{ .CacheSizeConstName }}; i++ {
		freeIndexes[i] = i
	}

	index := [{{ .IndexBucketsName }}]map[uint64]int{}
	for i := 0; i < {{ .IndexBucketsName }}; i++ {
		index[i] = make(map[uint64]int, {{ .CacheSizeConstName }}/{{ .IndexBucketsName }})
	}

	n := int32(len(freeIndexes))
	freeCount := &n

	c := &{{ .CacheType }}{
		&{{ .InnerCacheType }}{
			storage:         new([{{ .CacheSizeConstName }}]{{ .StoredValueType }}),
			index:           index,
			freeIndexes:     freeIndexes,
			freeCount:       freeCount,
			onClearing:      new(int32),
			freeIndexCh:     make(chan struct{}, {{ .FreeBatchSizeName }}),
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
func (c *{{ .CacheType }}) Set(key []byte, value *{{ .StoredType }}, expireSeconds int) error {
	if c.isClosed() {
		return CloseError{{ .StoredTypeSuffix }}
	}

	h := getHash{{ .StoredTypeSuffix }}(key)
	bucketIdx := getBucketIDs{{ .StoredTypeSuffix }}(h)
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
func (c *{{ .CacheType }}) Get(key []byte, value *{{ .StoredType }}) error {
	if c.isClosed() {
		return CloseError{{ .StoredTypeSuffix }}
	}

	if value == nil {
		return NilValueError{{ .StoredTypeSuffix }}
	}

	h := getHash{{ .StoredTypeSuffix }}(key)
	bucketIdx := getBucketIDs{{ .StoredTypeSuffix }}(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].RLock()
	valueIdx, ok := indexBucket[h]
	c.cache.indexLocks[bucketIdx].RUnlock()

	if !ok {
		return NotFoundError{{ .StoredTypeSuffix }}
	}

	c.cache.storageLocks[valueIdx].RLock()
	result := (*c.cache.storage)[valueIdx]
	c.cache.storageLocks[valueIdx].RUnlock()

	if result.expiredTime == {{ .DeletedValueFlagName }} {
		return NotFoundError{{ .StoredTypeSuffix }}
	}

	if (result.expiredTime - int(time.Now().Unix())) <= 0 {
		c.deleteItem(h, valueIdx)
		return NotFoundError{{ .StoredTypeSuffix }}
	}

	*value = result.v
	return nil
}

//Delete value by key
func (c *{{ .CacheType }}) Delete(key []byte) error {
	if c.isClosed() {
		return CloseError{{ .StoredTypeSuffix }}
	}

	h := getHash{{ .StoredTypeSuffix }}(key)
	bucketIdx := getBucketIDs{{ .StoredTypeSuffix }}(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].RLock()
	valueIdx, ok := indexBucket[h]
	c.cache.indexLocks[bucketIdx].RUnlock()

	c.cache.indexLocks[bucketIdx].Lock()
	delete(indexBucket, h)
	c.cache.indexLocks[bucketIdx].Unlock()

	c.cache.storageLocks[valueIdx].Lock()
	(*c.cache.storage)[valueIdx] = {{ .DeletedValueName }}
	(*c.cache.storage)[valueIdx].expiredTime = {{ .DeletedValueFlagName }}
	c.cache.storageLocks[valueIdx].Unlock()

	if !ok {
		return nil
	}

	c.pushFreeIndex(valueIdx)
	return nil
}

// deleteItem item by it bucket hash and index in bucket
func (c *{{ .CacheType }}) deleteItem(h uint64, valueIdx int) {
	bucketIdx := getBucketIDs{{ .StoredTypeSuffix }}(h)
	indexBucket := c.cache.index[bucketIdx]

	c.cache.indexLocks[bucketIdx].Lock()
	delete(indexBucket, h)
	c.cache.indexLocks[bucketIdx].Unlock()

	c.cache.storageLocks[valueIdx].Lock()
	(*c.cache.storage)[valueIdx] = {{ .DeletedValueName }}
	c.cache.storageLocks[valueIdx].Unlock()

	c.pushFreeIndex(valueIdx)
}

// get one index from storage to store new item
func (c *{{ .CacheType }}) popFreeIndex() int {
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

func (c *{{ .CacheType }}) forceClearCache() chan struct{} {
	if atomic.CompareAndSwapInt32(c.cache.onClearing, 0, 1) {
		c.cache.endClearingCh = make(chan struct{})
		c.cache.startClearingCh <- struct{}{}
	}
	return c.cache.endClearingCh
}

// push back index to mark it as free in cache storage
func (c *{{ .CacheType }}) pushFreeIndex(valueIdx int) {
	freeIdx := c.addFreeIndex()

	c.cache.freeIndexesLock.Lock()
	c.cache.freeIndexes[freeIdx] = valueIdx
	c.cache.freeIndexesLock.Unlock()
}

// increase freeIndexCount and returns new last free index
func (c *{{ .CacheType }}) addFreeIndex() int {
	return int(atomic.AddInt32(c.cache.freeCount, int32(1))) - 1 //Idx == new freeCount - 1 == old freeCount
}

// decrease freeIndexCount and returns new last free index
func (c *{{ .CacheType }}) removeFreeIndex(idx *int) int {
	*idx = int(atomic.AddInt32(c.cache.freeCount, int32(-1))) //Idx == new freeCount == old freeCount - 1
	if *idx < 0 {
		atomic.AddInt32(c.cache.freeCount, int32(1))
		return -1
	}
	return *idx
}

// increase freeIndexCount by N and returns new last free index
func (c *{{ .CacheType }}) addNFreeIndex(n int) int {
	return int(atomic.AddInt32(c.cache.freeCount, int32(n))) - 1 //Idx == new freeCount - 1 == old freeCount
}

func (c *{{ .CacheType }}) clearCache(startClearingCh chan struct{}) {
	var (
		freeIdx int

		nowTime time.Time
		now     int
		circle  int

		gcTicker = time.NewTicker({{ .GcTimeName }})

		currentChunk        int
		currentChunkIndexes [2]int
		indexInCacheArray   int

		iterateStoredValue {{ .StoredValueType }}

		freeIndexes = []int{}
		maxFreeIdx  int
	)

	// even for strange {{ .GcChunkSizeName }} chunks func guarantees that all indexes will present in result chunks
	chunks, _ := chunks{{ .StoredTypeSuffix }}({{ .CacheSizeConstName }}, {{ .GcChunkSizeName }})

	for {
		select {
		case <-startClearingCh:
			// forced gc
			if c.isClosed() {
				gcTicker.Stop()
				return
			}

			c.randomKeyCleanCacheStrategy(freeIdx)

			// Increase {{ .FreeBatchSizeName }} progressive
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

				if iterateStoredValue.expiredTime == {{ .DeletedValueFlagName }} {
					continue
				}

				if (iterateStoredValue.expiredTime - now) <= 0 {
					c.cache.storageLocks[indexInCacheArray].Lock()
					(*c.cache.storage)[indexInCacheArray] = {{ .DeletedValueName }}
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

func (*{{ .CacheType }}) setNewFreeBatchSize() {
	var freeBatchSizeDelta int = {{ .FreeBatchSizeName }} * {{ .AlphaName }} / 100
	if freeBatchSizeDelta < 1 {
		freeBatchSizeDelta = 1
	}

	{{ .FreeBatchSizeName }} += freeBatchSizeDelta
	if {{ .FreeBatchSizeName }} > ({{ .CacheSizeConstName }}*{{ .MaxFreeRatePercentName }})/100 {
		{{ .FreeBatchSizeName }} = ({{ .CacheSizeConstName }} * {{ .MaxFreeRatePercentName }}) / 100
	}

	if {{ .FreeBatchSizeName }} < 1 {
		{{ .FreeBatchSizeName }} = 1
	}
}

func (c *{{ .CacheType }}) randomKeyCleanCacheStrategy(freeIdx int) {
	i := 0

	for bucketIdx, bucket := range c.cache.index {
		c.cache.indexLocks[bucketIdx].Lock()

		for h, valueIdx := range bucket {
			delete(bucket, h)

			c.cache.storageLocks[valueIdx].Lock()
			if (*c.cache.storage)[valueIdx].expiredTime == {{ .DeletedValueFlagName }} {
				// trying to deleteItem deleted element in map
				c.cache.storageLocks[valueIdx].Unlock()
				continue
			}
			(*c.cache.storage)[valueIdx] = {{ .DeletedValueName }}
			c.cache.storageLocks[valueIdx].Unlock()

			freeIdx = c.addFreeIndex()

			c.cache.freeIndexesLock.Lock()
			c.cache.freeIndexes[freeIdx] = valueIdx
			c.cache.freeIndexesLock.Unlock()

			i++
			if i >= {{ .FreeBatchSizeName }} {
				break
			}
		}
		c.cache.indexLocks[bucketIdx].Unlock()
	}
}

//Flush cache. Can take a time
func (c *{{ .CacheType }}) Flush() error {
	if c.isClosed() {
		return CloseError{{ .StoredTypeSuffix }}
	}

	c.Close()

	newCache := newNice{{ .CacheType }}()

	// atomic store new cache
	oldPtr := (*unsafe.Pointer)(unsafe.Pointer(&c.cache))
	newUnsafe := unsafe.Pointer(newCache.cache)
	atomic.StorePointer(oldPtr, newUnsafe)

	return nil
}

//Len get stored elements count
func (c *{{ .CacheType }}) Len() int {
	if c.isClosed() {
		return 0
	}

	return {{ .CacheSizeConstName }} - int(atomic.LoadInt32(c.cache.freeCount))
}

//Close cache
func (c *{{ .CacheType }}) Close() {
	if c.isClosed() {
		return
	}

	close(c.cache.stop)
	atomic.StoreInt32(c.cache.isStopped, 1)
}

func (c *{{ .CacheType }}) isClosed() bool {
	return atomic.LoadInt32(c.cache.isStopped) == 1
}`
