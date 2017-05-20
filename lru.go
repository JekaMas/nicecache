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
	ChangeAction = Action(0)
	CreateAction = Action(1)
	DeleteAction = Action(2)
)

// LRU - это слайс, где индекс - это логарифм от количества вызовов элемента кэша с округлением вниз.
// внутри живет мапа, из которой при необходимости будем удалять элементы из нее
// если надо освободить какое-то количество элементов. то идем по слайсу и ищем ближайший индекс, в котором LruElement.count
// не равен нули - это будет бакет с элементами с наименьшим использованием
// мьютекс нужен, чтобы при расширении слайса мы не потеряли данные
type LRU struct {
	arr []LruElement // todo make it fixed size!
	*sync.RWMutex

	chEvent   chan LruEvent //неблокирующий канал событий кэша, если isBlocked выставлены, то события не принимаем - да, за небольшой период мы потеряем статистику, но зато не будет блокировки
	isBlocked *int32

	defaultSize int
}

func NewLru(size int) *LRU {
	isBlocked := int32(0)

	bucketsCount := LogHash(size/2) + 1

	arr := []LruElement{}

	lruStartBuckets := size / 4
	if lruStartBuckets <= 0 {
		lruStartBuckets = 1
	}

	for i := 0; i < bucketsCount; i++ {
		arr = append(arr, newLruElement(lruStartBuckets))
	}

	lock := sync.RWMutex{}

	lru := &LRU{
		arr:         arr,
		RWMutex:     &lock,
		chEvent:     make(chan LruEvent, size), //size/10
		isBlocked:   &isBlocked,
		defaultSize: size / 2,
	}

	go lru.do()

	return lru
}

type LruElement struct {
	index [indexBuckets]elem
	count *int32
}

type elem struct {
	m map[int]int //map[indexInSlice]frequencyValue
	*sync.RWMutex
	isBlocked *int32 // on write
}

func newLruElement(size int) LruElement {
	count := int32(0)

	index := [indexBuckets]elem{}

	for i := 0; i < indexBuckets; i++ {
		lock := sync.RWMutex{}
		notBlocked := int32(0)

		index[i] = elem{
			make(map[int]int, cacheSize/indexBuckets),
			&lock,
			&notBlocked,
		}
	}

	return LruElement{
		index: index,
		count: &count,
	}
}

type LruEvent struct {
	key    int
	action Action
}

func (l *LRU) Add(key int) {
	l.chEvent <- LruEvent{key, CreateAction}
}

func (l *LRU) add(key int) {
	buckets := l.arr[0]
	bucketIdx := getBucketIntIdx(key)
	bucket := buckets.index[bucketIdx]

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.Lock()
	bucket.m[key] = startCount
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	atomic.AddInt32(buckets.count, 1)

	return
}

// todo сделать горутину, принимающую события LruEvent
func (l *LRU) Change(key int) {
	l.chEvent <- LruEvent{key, ChangeAction}
}

func (l *LRU) change(key int, bucketsIdx, bucketIdx int) {
	buckets := l.arr[bucketsIdx]
	bucket := buckets.index[bucketIdx]

	if atomic.LoadInt32(l.isBlocked) == 1 || atomic.LoadInt32(bucket.isBlocked) == 1 {
		// try to get non blocking Get cache
		return
	}

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.RLock()
	if keyCount, ok := bucket.m[key]; ok {
		bucket.RUnlock()

		newBucketsIdx := LogHash(keyCount + 1)
		if newBucketsIdx != bucketsIdx {
			var newBuckets LruElement

			l.Delete(key)

			bucketsIdx = newBucketsIdx

			l.RLock()
			isIndexOverflow := (bucketsIdx == len(l.arr))
			l.RUnlock()

			if isIndexOverflow {
				// new bucket needed
				newBuckets = newLruElement(l.defaultSize / 2)

				l.Lock()
				l.arr = append(l.arr, newBuckets)
				l.Unlock()
			} else {
				l.RLock()
				newBuckets = l.arr[bucketsIdx]
				l.RUnlock()
			}

			newBucket := newBuckets.index[bucketIdx]

			atomic.StoreInt32(newBucket.isBlocked, 1)
			newBucket.Lock()
			newBucket.m[key] = keyCount + 1
			newBucket.Unlock()

			atomic.StoreInt32(newBucket.isBlocked, 0)
			atomic.AddInt32(newBuckets.count, 1)
		}

		bucket.RLock()
	}
	bucket.RUnlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	return
}

func (l *LRU) Delete(key int) {
	l.chEvent <- LruEvent{key, DeleteAction}
}

func (l *LRU) delete(key int, bucketIdx, bucketsIdx int) {
	if bucketsIdx == incorrectIndex {
		return
	}

	l.RLock()
	buckets := l.arr[bucketsIdx]
	l.Unlock()

	bucket := buckets.index[bucketIdx]

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.Lock()
	delete(bucket.m, key)
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	atomic.AddInt32(buckets.count, -1)
	return
}

func (l *LRU) MinBucket() int {
	l.RLock()

	for i, bucket := range l.arr {
		if atomic.LoadInt32(bucket.count) > 0 {
			l.RUnlock()
			return i
		}
	}

	l.RUnlock()

	return cacheIsEmpty
}

func (l *LRU) getBucketIdx(key int) (int, int) {
	bucketIdx := getBucketIntIdx(key)
	var bucket elem

	l.RLock()
	for i, buckets := range l.arr {
		l.RUnlock()

		bucket = buckets.index[bucketIdx]
		bucket.RLock()
		if _, ok := bucket.m[key]; ok {
			bucket.RUnlock()
			return i, bucketIdx
		}
		bucket.RUnlock()

		l.RLock()
	}
	l.RUnlock()

	return incorrectIndex, incorrectIndex
}

// DeleteCount deletes N records from least to more used - returns slice of int keys to delete
func (l *LRU) DeleteCount(count int) []int {
	// make channel instead
	keys := make([]int, 0, count)
	i := 0

	l.RLock()
	for _, buckets := range l.arr {

		for _, bucket := range buckets.index {
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
			atomic.AddInt32(buckets.count, -1)

			if i >= count {
				break
			}
		}
	}
	l.RUnlock()

	return keys
}

func (l *LRU) do() {
	var bucketsIdx int
	var bucketIdx int

	for event := range l.chEvent {
		bucketsIdx, bucketIdx = l.getBucketIdx(event.key)

		if bucketsIdx == incorrectIndex {
			l.add(event.key)
			continue
		}

		switch event.action {
		case ChangeAction:
			l.change(event.key, bucketsIdx, bucketIdx)
		case DeleteAction:
			l.delete(event.key, bucketsIdx, bucketIdx)
		}
	}
}

func (l *LRU) Close() {
	l.Lock()
	close(l.chEvent)
	l.Unlock()
}
