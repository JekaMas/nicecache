package nicecache

import (
	"strconv"
	"testing"
	"time"

	"github.com/JekaMas/pretty"
)

func TestCache_Get_OneKey(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

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

func TestCache_Get_OneKey_Repeat(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

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
	cache := NewNiceCache()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

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
	if err != NotFoundError {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Get_OneKey_TTL_ClearByGC(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

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
	if err != NotFoundError {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Get_OneKey_TTL_ClearByGC_OneKeyLive(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

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
	if err != NotFoundError {
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

func TestCache_Flush(t *testing.T) {

}

func TestCache_Delete(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

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
	if err != NotFoundError {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Delete_Repeat(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

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
	if err != NotFoundError {
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
	if err != NotFoundError {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Len_Zero(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Len_One(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 1 {
		t.Fatal(n)
	}
}

func TestCache_Len_Many(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	for i := 0; i < 3; i++ {
		key := []byte(strconv.Itoa(i))
		toStore := getTestValue()

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
	cache := NewNiceCache()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

	err := cache.Set(key, &toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	cache.Close()

	err = cache.Set(key, &toStore, longTime)
	if err != CloseError {
		t.Fatal(err)
	}

	if n := cache.Len(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Set_One_Repeat(t *testing.T) {
	cache := NewNiceCache()
	defer cache.Close()

	key := []byte(strconv.Itoa(1))
	toStore := getTestValue()

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

}

func TestCache_Set_Overflow_ManyTimes(t *testing.T) {

}

func TestCache_Set_Overflow_Flush_ManyTimes(t *testing.T) {

}

func TestCache_Set_Overflow_KeysExpired_Overflow(t *testing.T) {

}

func TestCache_Set_FullCache_RandomKeys(t *testing.T) {

}
