package nicecache

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"sync/atomic"
	"testing"
)

// TODO сделать тесты при кэше прогретом на 10, 30, 50, 100%
var testsize = (cacheSize * 1) / 100
var repeats = &testsize

const (
	longTime  = 600000
	shortTime = 1
)

func Benchmark_Cache_Nice_Set(b *testing.B) {
	cache := NewNiceCache()
	defer cache.Close()

	keys := make([][]byte, *repeats)
	for i := 0; i < *repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
	}
	toStore := testValues[0]

	var err error

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := cache.Set(keys[i%*repeats], &toStore, longTime)
		if e != nil {
			err = e
		}
	}
	b.StopTimer()

	if err != nil {
		fmt.Fprint(ioutil.Discard, "ERROR!!! SET ", err)
	}
}

func Benchmark_Cache_Nice_Set_Parallel(b *testing.B) {
	cache := NewNiceCache()
	defer cache.Close()

	keys := make([][]byte, *repeats)
	for i := 0; i < *repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
	}
	toStore := testValues[0]

	var i int32 = 0
	var err error

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddInt32(&i, 1)
			e := cache.Set(keys[int(i)%*repeats], &toStore, longTime)
			if e != nil {
				err = e
			}
		}
	})
	b.StopTimer()

	if err != nil {
		fmt.Fprint(ioutil.Discard, "ERROR!!! SET ", err)
	}
}

func Benchmark_Cache_Nice_Get(b *testing.B) {
	cache := NewNiceCache()
	defer cache.Close()

	keys := make([][]byte, *repeats)
	for i := 0; i < *repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
	}
	toStore := testValues[0]

	for i := 0; i < *repeats; i++ {
		err := cache.Set(keys[i], &toStore, longTime)
		if err != nil {
			fmt.Fprint(ioutil.Discard, "ERROR!!! SET: ", err)
		}
	}

	var (
		res TestValue
		err error
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = cache.Get(keys[i%*repeats], &res)
		if err != nil {
			fmt.Fprint(ioutil.Discard, "ERROR!!! GET ", err)
		}
	}
	b.StopTimer()

	fmt.Fprint(ioutil.Discard, res.ID)
}

func Benchmark_Cache_Nice_Get_Parallel(b *testing.B) {
	cache := NewNiceCache()
	defer cache.Close()

	keys := make([][]byte, *repeats)
	for i := 0; i < *repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
	}
	toStore := testValues[0]

	for i := 0; i < *repeats; i++ {
		err := cache.Set(keys[i], &toStore, longTime)
		if err != nil {
			fmt.Fprint(ioutil.Discard, "ERROR!!! SET ", err)
		}
	}

	var i int32 = 0
	var err error

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var res TestValue

			i := atomic.AddInt32(&i, 1)
			e := cache.Get(keys[int(i)%*repeats], &res)
			if e != nil {
				err = e
			}
		}
	})
	b.StopTimer()

	if err != nil {
		fmt.Fprint(ioutil.Discard, "ERROR!!! GET ", err)
	}
}

func Benchmark_Cache_Nice_SetAndGet(b *testing.B) {
	cache := NewNiceCache()
	defer cache.Close()

	keys := make([][]byte, *repeats)
	for i := 0; i < *repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
	}
	toStore := testValues[0]
	cache.Set(keys[0], &toStore, longTime)

	var res TestValue

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			cache.Set(keys[(i+1)%*repeats], &toStore, longTime)
			continue
		}
		cache.Get(keys[i%*repeats], &res)
	}
	b.StopTimer()

	fmt.Fprint(ioutil.Discard, res.ID)
}

func Benchmark_Cache_Nice_SetAndGet_Parallel(b *testing.B) {
	cache := NewNiceCache()
	defer cache.Close()

	keys := make([][]byte, *repeats)
	for i := 0; i < *repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
	}
	toStore := testValues[0]
	cache.Set(keys[0], &toStore, longTime)

	var i int32 = 0

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var res TestValue

			i := int(atomic.AddInt32(&i, 1))
			if i%2 == 0 {
				cache.Set(keys[(i+1)%*repeats], &toStore, longTime)
				continue
			}
			cache.Get(keys[i%*repeats], &res)
		}
	})
	b.StopTimer()
}

var (
	t   = true
	f   = false
	ids = []uint32{21, 12}
)

var testValues = []TestValue{
	{
		ID:           "1236564774641241249748124978917490",
		N:            6,
		Stat:         2,
		Published:    false,
		Deprecated:   &t,
		System:       43,
		Subsystem:    12,
		ParentID:     "4128934712974129075901235791027540",
		Name:         "some interesting test value!",
		Name2:        "once upon a time caches were fast and safe...",
		Description:  "yeah!!!",
		Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
		CreatedBy:    36,
		UpdatedBy:    &ids[0],
		List1:        []uint32{1, 2, 3, 4, 5},
		List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
		Items: []TestItem{
			{
				ID:           "1236564774641241249748124978917490",
				N:            6,
				Stat:         2,
				Published:    false,
				Deprecated:   &t,
				System:       43,
				Subsystem:    12,
				ParentID:     "4128934712974129075901235791027540",
				Name:         "some interesting test value!",
				Name2:        "once upon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &ids[0],
				List1:        []uint32{1, 2, 3, 4, 5},
				List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
			},
			{
				ID:           "1236564774641241249748124978917490",
				N:            6,
				Stat:         2,
				Published:    false,
				Deprecated:   &t,
				System:       43,
				Subsystem:    12,
				ParentID:     "4128934712974129075901235791027540",
				Name:         "some interesting test value!",
				Name2:        "once upon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &ids[0],
				List1:        []uint32{1, 2, 3, 4, 5},
				List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
			},
			{
				ID:           "1236564774641241249748124978917490",
				N:            6,
				Stat:         2,
				Published:    false,
				Deprecated:   &t,
				System:       43,
				Subsystem:    12,
				ParentID:     "4128934712974129075901235791027540",
				Name:         "some interesting test value!",
				Name2:        "once upon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &ids[0],
				List1:        []uint32{1, 2, 3, 4, 5},
				List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
			},
			{
				ID:           "1236564774641241249748124978917490",
				N:            6,
				Stat:         2,
				Published:    false,
				Deprecated:   &t,
				System:       43,
				Subsystem:    12,
				ParentID:     "4128934712974129075901235791027540",
				Name:         "some interesting test value!",
				Name2:        "once upon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &ids[0],
				List1:        []uint32{1, 2, 3, 4, 5},
				List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
			},
		},
	},

	{
		ID:           "1",
		N:            3,
		Stat:         4,
		Published:    false,
		Deprecated:   &f,
		System:       4543,
		Subsystem:    12,
		ParentID:     "3",
		Name:         "interesting test value!",
		Name2:        "once upon a Nicolaus Wirth creates Oberon...",
		Description:  "yeah!",
		Description2: "yeah!!",
		CreatedBy:    3,
		UpdatedBy:    &ids[1],
		List1:        []uint32{1, 3, 4, 5},
		List2:        []uint32{4, 3, 6, 8, 3, 4, 5, 6},
		Items: []TestItem{
			{
				ID:           "1236564774641241249748124978917490",
				N:            6,
				Stat:         2,
				Published:    false,
				Deprecated:   &t,
				System:       43,
				Subsystem:    12,
				ParentID:     "4128934712974129075901235791027540",
				Name:         "some interesting test value!",
				Name2:        "once upon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &ids[1],
				List1:        []uint32{1, 2, 3, 4, 5},
				List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
			},
			{
				ID:           "1236564774641241249748124978917490",
				N:            6,
				Stat:         2,
				Published:    false,
				Deprecated:   &t,
				System:       43,
				Subsystem:    12,
				ParentID:     "4128934712974129075901235791027540",
				Name:         "some interesting test value!",
				Name2:        "once upon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &ids[1],
				List1:        []uint32{1, 2, 3, 4, 5},
				List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
			},
			{
				ID:           "1236564774641241249748124978917490",
				N:            6,
				Stat:         2,
				Published:    false,
				Deprecated:   &t,
				System:       43,
				Subsystem:    12,
				ParentID:     "4128934712974129075901235791027540",
				Name:         "some interesting test value!",
				Name2:        "once upon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &ids[1],
				List1:        []uint32{1, 2, 3, 4, 5},
				List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
			},
			{
				ID:           "1236564774641241249748124978917490",
				N:            6,
				Stat:         2,
				Published:    false,
				Deprecated:   &t,
				System:       43,
				Subsystem:    12,
				ParentID:     "4128934712974129075901235791027540",
				Name:         "some interesting test value!",
				Name2:        "once upon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &ids[1],
				List1:        []uint32{1, 2, 3, 4, 5},
				List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
			},
		},
	},
}
