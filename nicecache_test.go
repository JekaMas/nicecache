package nicecache

import (
	"awg"
	"fmt"
	"testing"
)

const (
	keyLength = 1024
)

func Benchmark_Cache_Circle_Set(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, b.N, b.N)
	values := make([]TestValue, b.N, b.N)
	for i := 0; i < b.N; i++ {
		key := make([]byte, keyLength, keyLength)
		key[i%keyLength] = 'a'
		keys[i] = key

		values[i] = getTestValue()
	}

	var err error
	b.ReportAllocs()
	b.StartTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := cache.Set(keys[i], values[i], 180)
		if e != nil {
			err = e
		}
	}
	b.StopTimer()

	if err != nil {
		fmt.Println("ERROR!!! SET ", err)
	}
}

func Benchmark_Cache_Circle_Set_Parallel(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, b.N, b.N)
	values := make([]TestValue, b.N, b.N)
	for i := 0; i < b.N; i++ {
		key := make([]byte, keyLength, keyLength)
		key[i%keyLength] = 'a'
		keys[i] = key

		values[i] = getTestValue()
	}

	wg := &awg.AdvancedWaitGroup{}
	wg.SetCapacity(4)
	b.ReportAllocs()
	b.StartTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		i := i
		wg.Add(func() error {
			cache.Set(keys[i], values[i], 180)
			return nil
		})

	}
	wg.Start()
	b.StopTimer()
}

func Benchmark_Cache_Circle_Get(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, b.N, b.N)
	values := make([]TestValue, b.N, b.N)
	for i := 0; i < b.N; i++ {
		key := make([]byte, keyLength, keyLength)
		key[i%keyLength] = 'a'
		keys[i] = key

		values[i] = getTestValue()
	}

	for i := 0; i < b.N; i++ {
		err := cache.Set(keys[i], values[i], 180)
		if err != nil {
			fmt.Println("ERROR!!! SET: ", err)
		}
	}

	var (
		res TestValue
		err error
	)
	b.ReportAllocs()
	b.StartTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err = cache.Get(keys[i])
		if err != nil {
			fmt.Println("ERROR!!! GET ", err)
		}
	}
	b.StopTimer()

	fmt.Println(res.ID)
}

func Benchmark_Cache_Circle_Get_Parallel(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, b.N, b.N)
	values := make([]TestValue, b.N, b.N)
	for i := 0; i < b.N; i++ {
		key := make([]byte, keyLength, keyLength)
		key[i%keyLength] = 'a'
		keys[i] = key

		values[i] = getTestValue()
	}

	for i := 0; i < b.N; i++ {
		err := cache.Set(keys[i], values[i], 180)

		if err != nil {
			fmt.Println("ERROR!!! SET ", err)
		}
	}

	wg := &awg.AdvancedWaitGroup{}
	wg.SetCapacity(4)
	b.ReportAllocs()
	b.StartTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		i := i
		wg.Add(func() error {
			res, err := cache.Get(keys[i])
			_ = res
			return err
		})

	}
	wg.Start()
	b.StopTimer()

	err := wg.GetLastError()
	if err != nil {
		fmt.Println("ERROR!!! GET ", err)
	}
}

func Benchmark_Cache_Circle_SetAndGet(b *testing.B) {
	cache := NewNiceCache()

	keys := make([][]byte, b.N, b.N)
	values := make([]TestValue, b.N, b.N)
	for i := 0; i < b.N; i++ {
		key := make([]byte, keyLength, keyLength)
		key[i%keyLength] = 'a'
		keys[i] = key

		values[i] = getTestValue()
	}

	var res TestValue
	b.ReportAllocs()
	b.StartTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(keys[(i+1)%b.N], values[(i+1)%b.N], 180)
		res, _ = cache.Get(keys[i])
	}
	b.StopTimer()

	fmt.Println(res.ID)
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
	}
}
