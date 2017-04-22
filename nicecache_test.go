package nicecache

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"sync/atomic"
	"testing"
)

var repeats = freeBatchSize - 1

func Benchmark_Cache_Circle_Set(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, repeats)
	values := make([]TestValue, repeats)
	for i := 0; i < repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
		values[i] = getTestValue()
	}

	var err error
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := cache.Set(keys[i%repeats], values[i%repeats], 180)
		if e != nil {
			err = e
		}
	}
	b.StopTimer()

	if err != nil {
		fmt.Fprint(ioutil.Discard, "ERROR!!! SET ", err)
	}
}

func Benchmark_Cache_Circle_Set_Parallel(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, repeats)
	values := make([]TestValue, repeats)
	for i := 0; i < repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
		values[i] = getTestValue()
	}

	var i int32 = 0
	var err error
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddInt32(&i, 1)
			e := cache.Set(keys[int(i)%repeats], values[int(i)%repeats], 180)
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

func Benchmark_Cache_Circle_Get(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, repeats)
	values := make([]TestValue, repeats)
	for i := 0; i < repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
		values[i] = getTestValue()
	}

	for i := 0; i < repeats; i++ {
		err := cache.Set(keys[i], values[i], 180)
		if err != nil {
			fmt.Fprint(ioutil.Discard, "ERROR!!! SET: ", err)
		}
	}

	var (
		res TestValue
		err error
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err = cache.Get(keys[i%repeats])
		if err != nil {
			fmt.Fprint(ioutil.Discard, "ERROR!!! GET ", err)
		}
	}
	b.StopTimer()

	fmt.Fprint(ioutil.Discard, res.ID)
}

func Benchmark_Cache_Circle_Get_Parallel(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, repeats)
	values := make([]TestValue, repeats)
	for i := 0; i < repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
		values[i] = getTestValue()
	}

	for i := 0; i < repeats; i++ {
		err := cache.Set(keys[i], values[i], 180)
		if err != nil {
			fmt.Fprint(ioutil.Discard, "ERROR!!! SET ", err)
		}
	}

	var i int32 = 0
	var err error
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddInt32(&i, 1)
			_, e := cache.Get(keys[int(i)%repeats])
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

func Benchmark_Cache_Circle_SetAndGet(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, repeats)
	values := make([]TestValue, repeats)
	for i := 0; i < repeats; i++ {
		keys[i] = []byte(strconv.Itoa(i))
		values[i] = getTestValue()
	}
	cache.Set(keys[0], values[0], 180)

	var res TestValue
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(keys[(i+1)%repeats], values[(i+1)%repeats], 180)
		res, _ = cache.Get(keys[i%repeats])
	}
	b.StopTimer()

	fmt.Fprint(ioutil.Discard, res.ID)
}

func getTestValue() TestValue {
	t := true
	id := uint32(21)

	return TestValue{
		ID:           "1236564774641241249748124978917490",
		N:            6,
		Stat:         2,
		Published:    false,
		Deprecated:   &t,
		System:       43,
		Subsystem:    12,
		ParentID:     "4128934712974129075901235791027540",
		Name:         "some interesting test value!",
		Name2:        "once apon a time caches were fast and safe...",
		Description:  "yeah!!!",
		Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
		CreatedBy:    36,
		UpdatedBy:    &id,
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
				Name2:        "once apon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &id,
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
				Name2:        "once apon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &id,
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
				Name2:        "once apon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &id,
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
				Name2:        "once apon a time caches were fast and safe...",
				Description:  "yeah!!!",
				Description2: "yeah!!! yeah!!! yeah!!! yeah!!! yeah!!!",
				CreatedBy:    36,
				UpdatedBy:    &id,
				List1:        []uint32{1, 2, 3, 4, 5},
				List2:        []uint32{6, 4, 3, 6, 8, 3, 4, 5, 6},
			},
		},
	}
}
