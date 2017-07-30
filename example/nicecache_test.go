package example

import (
	"strconv"
	"testing"
	"time"

	"github.com/JekaMas/pretty"
)

func TestCache_Get_OneKeyExists(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func TestCache_Get_OneKey_NotExists(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	key = []byte(strconv.Itoa(2))
	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}
}

func TestCache_Get_OneKey_NilValue(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	err = cache.Get(key, nil)
	if err != NilValueErrorTestValue {
		t.Fatal(err)
	}
}

func TestCache_Get_OneKey_Repeat(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	//retry
	gotValue = TestValue{}
	err = cache.Get(key, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func TestCache_Get_OneKey_TTL_ClearOnRead(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, shortTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}

	//retry after TTL
	time.Sleep(shortTime * 2 * time.Second)

	gotValue = TestValue{}
	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Get_OneKey_TTL_ClearByGC(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, shortTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}

	//retry after TTL
	time.Sleep(shortTime * 120 * time.Second)

	gotValue = TestValue{}
	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Get_OneKey_TTL_ClearByGC_OneKeyLive(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, shortTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}

	//second key
	toStore = testValues[1]
	key2 := []byte(strconv.Itoa(2))

	err = cache.Set(key2, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue = TestValue{}
	err = cache.Get(key2, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := cache.Len(); n != 2 {
		t.Fatal(n)
	}

	//retry after TTL
	time.Sleep(shortTime * 120 * time.Second)

	// expired key
	gotValue = TestValue{}
	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}

	// live key
	gotValue = TestValue{}
	err = cache.Get(key2, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}
}

func TestCache_Get_OneKey_TTL_ClearByGC_OneKeyLive_ManyCircles(t *testing.T) {

}

func TestCache_Get_ManyKeys(t *testing.T) {

}

func TestCache_Get_ManyKeys_TTL(t *testing.T) {

}

func TestCache_Get_Set_LongTime_CheckMemoryLeak(t *testing.T) {

}

func TestCache_Get_Close(t *testing.T) {
	cache := NewNiceCacheTestValue()
	cache.Close()

	gotValue := TestValue{}
	err := cache.Get([]byte{1}, &gotValue)
	if err != CloseErrorTestValue {
		t.Fatal(err)
	}
}

func TestCache_Flush(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	err = cache.Flush()
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Flush_Close(t *testing.T) {
	cache := NewNiceCacheTestValue()
	cache.Close()

	err := cache.Flush()
	if err != CloseErrorTestValue {
		t.Fatal(err)
	}
}

func TestCache_Flush_Repeat(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	err = cache.Flush()
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}

	// repeat
	err = cache.Flush()
	if err != nil {
		t.Fatal(err)
	}

	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Delete(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	err = cache.Delete(key)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Delete_Close(t *testing.T) {
	cache := NewNiceCacheTestValue()
	cache.Close()

	err := cache.Delete([]byte{1})
	if err != CloseErrorTestValue {
		t.Fatal(err)
	}
}

func TestCache_Delete_Repeat(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	err = cache.Delete(key)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}

	// repeat
	err = cache.Delete(key)
	if err != nil {
		t.Fatal(err)
	}

	err = cache.Get(key, &gotValue)
	if err != NotFoundErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Len_Zero(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Len_One(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}
}

func TestCache_Len_Close(t *testing.T) {
	cache := NewNiceCacheTestValue()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}

	cache.Close()

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Len_Many(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	for i := 0; i < 2; i++ {
		key := []byte(strconv.Itoa(i))
		toStore := testValues[i]

		err := cache.Set(key, &toStore, longTime)
		if err != nil {
			t.Fatal(err)
		}

		if n := cache.Len(); n != i+1 {
			t.Fatal(n)
		}
	}
}

func TestCache_Close(t *testing.T) {
	cache := NewNiceCacheTestValue()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	cache.Close()

	err = cache.Set(key, &toStore, longTime)
	if err != CloseErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Close_Repeat(t *testing.T) {
	cache := NewNiceCacheTestValue()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	cache.Close()
	cache.Close()

	err = cache.Set(key, &toStore, longTime)
	if err != CloseErrorTestValue {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Set_One_Repeat(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}

	//retry
	toStore = testValues[1]
	err = cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue = TestValue{}
	err = cache.Get(key, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}
}

func TestCache_Set_Overflow(t *testing.T) {
	cache := NewNiceCacheTestValue()
	defer cache.Close()

	for i := 0; i < 2*cacheSizeTestValue; i++ {
		key := []byte(strconv.Itoa(i))
		toStore := testValues[0]

		err := cache.Set(key, &toStore, longTime)
		if err != nil {
			t.Fatal(err)
		}
	}

	key := []byte(strconv.Itoa(1))
	toStore := testValues[0]

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue := TestValue{}
	err = cache.Get(key, &gotValue)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := cache.Len(); n <= int(cacheSizeTestValue*(1 - float64(freeBatchSizeTestValue+1)/float64(100))) {
		t.Fatal(n)
	}

	if n := cache.Len(); n > cacheSizeTestValue {
		t.Fatal(n)
	}
}

func TestCache_Set_Overflow_KeysExpired_Overflow(t *testing.T) {

}

func TestCache_Set_FullCache_RandomKeys(t *testing.T) {

}
