package nicecache

import (
	"sync"
	"sync/atomic"
)

const (
	startCount = 0

	incorrectIndex = -1
	cacheIsEmpty   = -1
)

type Action uint8

const (
	ChangeAction = Action(0) // and add
	DeleteAction = Action(1)

	lruShards = indexBuckets * indexBuckets
)

// all keys are shards over indexBuckets^2 maps

// LRU - это слайс, где индекс - это логарифм от количества вызовов элемента кэша с округлением вниз.
// внутри живет мапа, из которой при необходимости будем удалять элементы из нее
// если надо освободить какое-то количество элементов. то идем по слайсу и ищем ближайший индекс, в котором LruElement.count
// не равен нули - это будет бакет с элементами с наименьшим использованием
// мьютекс нужен, чтобы при расширении слайса мы не потеряли данные
type LRU struct {
	shards  *[lruShards]*LRUShard
	chEvent chan LruEvent //неблокирующий канал событий кэша, если isBlocked выставлены, то события не принимаем - да, за небольшой период мы потеряем статистику, но зато не будет блокировки
}

func NewLru(size int) *LRU {
	shards := [lruShards]*LRUShard{}

	chEvent := make(chan LruEvent, size/5) //todo tune it to consume less memory

	for i := 0; i < len(shards); i++ {
		shards[i] = NewLRUShard(cacheSize / indexBuckets)
	}

	lru := &LRU{&shards, chEvent}
	for i := 0; i < 5; i++ {
		go lru.do() //todo: tune number of goroutines
	}

	return lru
}

type LruElement struct {
	m map[int]int //map[indexInSlice]frequencyValue
	*sync.RWMutex
	isBlocked *int32 // on write
	count     *int32
}

func newLruElement() LruElement {
	count := int32(0)
	lock := sync.RWMutex{}
	notBlocked := int32(0)

	return LruElement{
		make(map[int]int),
		&lock,
		&notBlocked,
		&count,
	}
}

type LruEvent struct {
	key    int
	action Action
}

// todo сделать горутину, принимающую события LruEvent
func (l *LRU) Change(key int) {
	l.chEvent <- LruEvent{key, ChangeAction}
}

func (l *LRU) Delete(key int) {
	l.chEvent <- LruEvent{key, DeleteAction}
}

// DeleteCount deletes N records from least to more used - returns slice of int keys to delete
func (l *LRU) DeleteCount(count int) []int {
	// make channel instead
	keys := make([]int, 0, count)
	i := 0

	for _, shard := range l.shards {
		//todo возможно делать в несколько горутин
		shard.RLock()
		for _, bucket := range shard.arr {
			atomic.StoreInt32(bucket.isBlocked, 1)
			bucket.Lock()
			for key := range bucket.m {
				delete(bucket.m, key)
				keys = append(keys, key)

				i++
				if i >= count {
					break
				}
			}
			bucket.Unlock()

			atomic.StoreInt32(bucket.isBlocked, 0)
			atomic.AddInt32(bucket.count, -1)

			if i >= count {
				break
			}
		}
		shard.RUnlock()
	}
	return keys
}

var debugMap = make(map[int]int)
var lock = sync.RWMutex{}

func (l *LRU) do() {
	var (
		shardIdx  int
		bucketIdx int
		shard     *LRUShard
	)

	for event := range l.chEvent {
		shardIdx = getLruShardIdx(event.key)
		shard = l.shards[shardIdx]
		bucketIdx = shard.getBucketIdx(event.key)

		switch event.action {
		case ChangeAction:
			shard.change(event.key, bucketIdx)
		case DeleteAction:
			shard.delete(event.key, bucketIdx)
		}
	}
}

func (l *LRU) Close() {
	close(l.chEvent)

	/*
		lock.RLock()
		spew.Dump(len(debugMap))
		lock.RUnlock()
	*/
	/*
		//todo сделать сбор использования шард и бакетов + их вывод для отладки
		for i, shard := range l.shards {
			nonEmpty := int(0)
			for j, bucket := range shard.arr {
				if atomic.LoadInt32(bucket.count) > 0 {
					nonEmpty++
				}
				fmt.Printf("Shard %v, bucket %v - %v\n", i, j, atomic.LoadInt32(bucket.count))
			}
			fmt.Printf("Shard %v has %v non empty buckets\n__________________________________________________\n\n", i, nonEmpty)
		}
	*/
}

type LRUShard struct {
	arr []LruElement // todo make it fixed size!
	*sync.RWMutex

	isBlocked *int32
}

func NewLRUShard(size int) *LRUShard {
	isBlocked := int32(0)

	bucketsCount := LogHash(size/2) + 1

	arr := []LruElement{}

	for i := 0; i < bucketsCount; i++ {
		arr = append(arr, newLruElement())
	}

	lock := sync.RWMutex{}

	lruShard := &LRUShard{
		arr:       arr,
		RWMutex:   &lock,
		isBlocked: &isBlocked,
	}

	return lruShard
}

func (l *LRUShard) add(key int) {
	bucket := l.arr[0]

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.Lock()
	bucket.m[key] = startCount
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)
	atomic.AddInt32(bucket.count, 1)

	return
}

//todo rename receiver
func (l *LRUShard) change(key int, bucketIdx int) {
	if bucketIdx == incorrectIndex {
		l.add(key)
		return
	}

	bucket := l.arr[bucketIdx]
	if !atomic.CompareAndSwapInt32(bucket.isBlocked, 0, 1) || atomic.LoadInt32(l.isBlocked) == 1 {
		// try to get non blocking Get cache
		return
	}

	bucket.Lock()
	keyCount, _ := bucket.m[key]
	bucket.m[key] = keyCount+1
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	newBucketIdx := LogHash(keyCount + 1)
	if newBucketIdx != bucketIdx {
		// need to move key to bucket with more used keys
		var newBucket LruElement

		l.RLock()
		isIndexOverflow := (newBucketIdx == len(l.arr))
		l.RUnlock()

		if isIndexOverflow {
			// new bucket needed
			newBucket = newLruElement()

			l.Lock()
			l.arr = append(l.arr, newBucket)
			l.Unlock()
		} else {
			l.RLock()
			newBucket = l.arr[newBucketIdx]
			l.RUnlock()
		}

		atomic.StoreInt32(newBucket.isBlocked, 1)
		newBucket.Lock()
		newBucket.m[key] = keyCount + 1
		newBucket.Unlock()

		atomic.StoreInt32(newBucket.isBlocked, 0)
		atomic.AddInt32(newBucket.count, 1)

		//delete key from old bucket
		l.delete(key, bucketIdx)
	}

	return
}

func (l *LRUShard) delete(key int, bucketIdx int) {
	if bucketIdx == incorrectIndex {
		return
	}

	l.RLock()
	bucket := l.arr[bucketIdx]
	l.RUnlock()

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.Lock()
	delete(bucket.m, key)
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	atomic.AddInt32(bucket.count, -1)
	return
}

func (l *LRUShard) getBucketIdx(key int) int {
	l.RLock()
	for i, bucket := range l.arr {
		bucket.RLock()
		_, ok := bucket.m[key]
		bucket.RUnlock()
		if ok {
			return i
		}
	}
	l.RUnlock()

	return incorrectIndex
}
