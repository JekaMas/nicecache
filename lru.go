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
	arr []*LruElement // use
	*sync.RWMutex

	chEvent   chan LruEvent //неблокирующий канал событий кэша, если isBlocked выставлены, то события не принимаем - да, за небольшой период мы потеряем статистику, но зато не будет блокировки
	isBlocked *int32

	defaultSize int
}

func NewLru(size int) *LRU {
	isBlocked := int32(0)

	bucketsCount := LogHash(size/2) + 1

	arr := []*LruElement{}

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
	m map[int]int //map[indexInSlice]frequencyValue
	sync.RWMutex
	count *int32

	isBlocked *int32 // on write
}

func newLruElement(size int) *LruElement {
	isBlocked := int32(0)
	count := int32(0)

	return &LruElement{
		m:         make(map[int]int, size),
		count:     &count,
		isBlocked: &isBlocked,
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
	l.RLock()
	bucket := l.arr[0]

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.Lock()
	bucket.m[key] = startCount
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	l.RUnlock()

	atomic.AddInt32(bucket.count, 1)

	return
}

// todo сделать горутину, принимающую события LruEvent
func (l *LRU) Change(key int) {
	l.chEvent <- LruEvent{key, ChangeAction}
}

func (l *LRU) change(key int, bucketIdx int) {
	bucket := l.arr[bucketIdx]

	if atomic.LoadInt32(l.isBlocked) == 1 || atomic.LoadInt32(bucket.isBlocked) == 1 {
		// try to get non blocking Get cache
		return
	}

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.RLock()
	if keyCount, ok := bucket.m[key]; ok {
		bucket.RUnlock()

		newBucketIdx := LogHash(keyCount + 1)
		if newBucketIdx != bucketIdx {
			var newBucket *LruElement

			l.Delete(key)

			bucketIdx = newBucketIdx

			if bucketIdx == len(l.arr) {
				// new bucket needed
				newBucket = newLruElement(l.defaultSize / 2)

				l.Lock()
				l.arr = append(l.arr, newBucket)
				l.Unlock()
			} else {
				newBucket = l.arr[bucketIdx]
			}

			atomic.StoreInt32(newBucket.isBlocked, 1)
			newBucket.Lock()
			newBucket.m[key] = keyCount + 1
			bucket.Unlock()

			atomic.StoreInt32(newBucket.isBlocked, 0)
			atomic.AddInt32(newBucket.count, 1)
		}
	}
	bucket.RUnlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	l.RUnlock()

	return
}

func (l *LRU) Delete(key int) {
	l.chEvent <- LruEvent{key, DeleteAction}
}

func (l *LRU) delete(key int, bucketIdx int) {
	if bucketIdx != incorrectIndex {
		l.RLock()
		if bucketIdx >= len(l.arr) {
			l.RUnlock()
			return
		}
	} else {
		l.RLock()

		for i, bucket := range l.arr {
			bucket.RLock()
			if _, ok := bucket.m[key]; ok {
				bucket.RUnlock()
				bucketIdx = i
				break
			}
			bucket.RUnlock()
		}
	}

	if bucketIdx <= incorrectIndex {
		// incorrect bucket index or key not found
		l.RUnlock()
		return
	}

	bucket := l.arr[bucketIdx]

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.Lock()
	delete(bucket.m, key)
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	l.RUnlock()

	atomic.AddInt32(bucket.count, -1)
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

func (l *LRU) getBucketIdx(key int) int {
	l.RLock()
	for i, bucket := range l.arr {
		bucket.RLock()
		if _, ok := bucket.m[key]; ok {
			bucket.RUnlock()
			l.RUnlock()

			return i
		}
		bucket.RUnlock()
	}
	l.RUnlock()

	return incorrectIndex
}

// DeleteCount deletes N records from least to more used - returns slice of int keys to delete
func (l *LRU) DeleteCount(count int) []int {
	// make channel instead
	keys := make([]int, 0, count)
	i := 0

	l.RLock()
	for _, bucket := range l.arr {
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
	l.RUnlock()

	return keys
}

func (l *LRU) do() {
	var bucketIdx int

	for event := range l.chEvent {
		bucketIdx = l.getBucketIdx(event.key)

		if bucketIdx == incorrectIndex {
			l.add(event.key)
			continue
		}

		switch event.action {
		case ChangeAction:
			l.change(event.key, bucketIdx)
		case DeleteAction:
			l.delete(event.key, bucketIdx)
		}
	}
}

func (l *LRU) Close() {
	l.Lock()
	close(l.chEvent)
	l.Unlock()
}
