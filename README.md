# Nicecache

This is fastest golang cache*
* statically typed
* minimum read-write locks
* interface{}-free
* zero allocation for read and write


#### Note
Im really need you advices and usage experience, please feel free to write me you issues, feature requests and use cases.


## Usage

First of all you need to generate nicecache files.
You could create cache file in the same package as cached type:
```go
//go:generate nicecache -type=TestValue -cacheSize=10000000
type TestValue struct {
    ...
}
```

Or create cache file in other package:
```go
//go:generate nicecache -type=TestItem -cacheSize=10000000 -cachePackage=nicecache/example/repository
type TestItem struct {
    ...
}
```

```go
// create and init cache
cache := NewNiceCacheTestValue()
defer cache.Close() // close when needed (app exit for example)

key := []byte("product_category_123")  // sync.Pool can be used
toStore := TestType{...}

err := cache.Set(key, &toStore, longTime)
if err != nil {
	//...
}

gotValue := TestType{} // sync.Pool can be used
err = cache.Get(key, &gotValue)
if err != nil {
	//...
}
```


## Example

Example [is here.](example)


## Benchmarks

Cache size 1.000.000 items, performance on 4 GOMAXPROCS:
```
Benchmark_Cache_Nice_Set-4                      100000000              132 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-4             200000000               88.4 ns/op             0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-4                      100000000              129 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-4             200000000               61.9 ns/op             0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-4                100000000              121 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet_Parallel-4       100000000              166 ns/op               0 B/op          0 allocs/op
     
Benchmark_Chunked_MapStringInterface_Set-4                     2000000               894 ns/op             219 B/op          1 allocs/op
Benchmark_Chunked_MapStringInterface_Set_Parallel-4            1000000              1151 ns/op             247 B/op          1 allocs/op
Benchmark_Chunked_MapStringInterface_Get-4                     1000000              1952 ns/op             336 B/op          2 allocs/op
Benchmark_Chunked_MapStringInterface_Get_Parallel-4            1000000              2732 ns/op             336 B/op          2 allocs/op
Benchmark_Chunked_MapStringInterface_SetAndGet-4               1000000              3247 ns/op             583 B/op          3 allocs/op
```

Cache size 10.000.000 items, performance on 4 GOMAXPROCS:
```
Benchmark_Cache_Nice_Set-4                      100000000              196 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-4             200000000               96.2 ns/op             0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-4                      100000000              199 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-4             200000000               82.9 ns/op             0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-4                100000000              158 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet_Parallel-4       100000000              186 ns/op               0 B/op          0 allocs/op
    
Benchmark_Chunked_MapStringInterface_Set-4                     1000000              2766 ns/op             761 B/op          5 allocs/op
Benchmark_Chunked_MapStringInterface_Set_Parallel-4            1000000              3243 ns/op             761 B/op          5 allocs/op
Benchmark_Chunked_MapStringInterface_Get-4                     1000000              1925 ns/op             336 B/op          2 allocs/op
Benchmark_Chunked_MapStringInterface_Get_Parallel-4            1000000              2345 ns/op             336 B/op          2 allocs/op
Benchmark_Chunked_MapStringInterface_SetAndGet-4                300000              5638 ns/op            1078 B/op          7 allocs/op
```

Cache size 1.000.000 items, performance on 8 GOMAXPROCS:
```
Benchmark_Cache_Nice_Set-8                      100000000              131 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-8             200000000               62.4 ns/op             0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-8                      100000000              131 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-8             300000000               42.4 ns/op             0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-8                100000000              122 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet_Parallel-8       100000000              132 ns/op               0 B/op          0 allocs/op
```

Cache size 10.000.000 items, performance on 8 GOMAXPROCS:
```
Benchmark_Cache_Nice_Set-8                      100000000              191 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-8             200000000               65.5 ns/op             0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-8                      100000000              207 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-8             300000000               51.7 ns/op             0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-8                100000000              155 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet_Parallel-8       100000000              143 ns/op               0 B/op          0 allocs/op
```


###### Remarks:
*as far as i know :)