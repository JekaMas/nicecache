package nicecache

import (
	"sync"
	"sync/atomic"
)

const (
	startBucket = 0

	incorrectIndex = -1
	cacheIsEmpty   = -1
)

type Action uint8

const (
	ChangeAction = Action(0)
	CreateAction = Action(1)
	DeleteAction = Action(2)
)

const defaultMapSize = 5000000

// LRU - это слайс, где индекс - это логарифм от количества вызовов элемента кэша с округлением вниз.
// внутри живет мапа, из которой при необходимости будем удалять элементы из нее
// если надо освободить какое-то количество элементов. то идем по слайсу и ищем ближайший индекс, в котором LruElement.count
// не равен нули - это будет бакет с элементами с наименьшим использованием
// мьютекс нужен, чтобы при расширении слайса мы не потеряли данные
type LRU struct {
	arr []*LruElement // use
	sync.RWMutex

	chEvent   chan LruEvent //неблокирующий канал событий кэша, если isBlocked выставлены, то события не принимаем - да, за небольшой период мы потеряем статистику, но зато не будет блокировки
	isBlocked *int32

	defaultSize int
}

func NewLru(size int) *LRU {
	isBlocked := int32(0)

	bucketsCount := LogHash(size/2) + 1

	arr := make([]*LruElement, bucketsCount)
	for i := range arr {
		arr[i] = newLruElement(size/4)
	}

	return &LRU{
		arr:       arr,
		//chEvent:   make(chan LruEvent, size),
		isBlocked: &isBlocked,
		defaultSize: size/2,
	}
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
	Key    int
	Action Action
}

func (l *LRU) Add(key int) int {
	l.Delete(key)

	bucket := l.arr[0]

	l.RLock()

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.Lock()
	bucket.m[key] = startBucket
	atomic.AddInt32(bucket.count, 1)
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	l.RUnlock()

	return startBucket
}

// todo сделать горутину, принимающую события LruEvent
// подумать, сохранять ли у самого элемента storedValue в кэше его bucket в LRU - думаю, да сохранять
func (l *LRU) Change(key int, bucketIdx int) (int, bool) {
	var isBucketChanged bool

	l.RLock()
	if bucketIdx <= incorrectIndex || bucketIdx >= len(l.arr) {
		// incorrect bucket index
		l.RUnlock()
		return incorrectIndex, isBucketChanged
	}

	bucket := l.arr[bucketIdx]

	if atomic.LoadInt32(l.isBlocked) == 1 || atomic.LoadInt32(bucket.isBlocked) == 1 {
		// try to get non blocking Get cache
		return bucketIdx, isBucketChanged
	}

	atomic.StoreInt32(bucket.isBlocked, 1)
	bucket.Lock()
	if keyCount, ok := bucket.m[key]; ok {
		newBucketIdx := LogHash(keyCount + 1)
		if newBucketIdx != bucketIdx {
			var newBucket *LruElement

			isBucketChanged = true

			l.Delete(key, bucketIdx)

			bucketIdx = newBucketIdx

			if bucketIdx == len(l.arr) {
				// new bucket needed
				newBucket = newLruElement(l.defaultSize/2)

				l.Lock()
				l.arr = append(l.arr, newBucket)
				l.Unlock()
			} else {
				newBucket = l.arr[bucketIdx]
			}

			atomic.StoreInt32(newBucket.isBlocked, 1)
			newBucket.Lock()
			newBucket.m[key] = keyCount + 1
			atomic.AddInt32(newBucket.count, 1)
			bucket.Unlock()
			atomic.StoreInt32(newBucket.isBlocked, 0)
		}
	}
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	l.RUnlock()

	return bucketIdx, isBucketChanged
}

func (l *LRU) Delete(key int, bucketIdxs ...int) {
	var bucketIdx int = incorrectIndex

	if len(bucketIdxs) > 0 {
		bucketIdx = bucketIdxs[0]

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
	atomic.AddInt32(bucket.count, -1)
	bucket.Unlock()
	atomic.StoreInt32(bucket.isBlocked, 0)

	l.RUnlock()
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

// DeleteCount deletes N records from least to more used - returns slice of int keys to delete
func (l *LRU) DeleteCount(count int) []int {
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
		atomic.AddInt32(bucket.count, -1)
		bucket.Unlock()
		atomic.StoreInt32(bucket.isBlocked, 0)

		if i >= count {
			break
		}
	}
	l.RUnlock()

	return keys
}
