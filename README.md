размер кэша 1024&times;1024
```
Benchmark_Cache_Circle_Set-4                     3000000               438 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_Set_Parallel-4            2000000              1429 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_Get-4                     5000000               406 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_Get_Parallel-4           10000000               303 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_SetAndGet-4               3000000               572 ns/op               0 B/op          0 allocs/op
     
Benchmark_Cache_Struct_Set-4                     2000000               894 ns/op             219 B/op          1 allocs/op
Benchmark_Cache_Struct_Set_Parallel-4            1000000              1151 ns/op             247 B/op          1 allocs/op
Benchmark_Cache_Struct_Get-4                     1000000              1952 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_Get_Parallel-4            1000000              2732 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_SetAndGet-4               1000000              3247 ns/op             583 B/op          3 allocs/op
```

размер кэша 1024&times;1024&times;10
```
Benchmark_Cache_Circle_Set-4                     2000000               648 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_Set_Parallel-4            1000000              1402 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_Get-4                     3000000               411 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_Get_Parallel-4           10000000               177 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_SetAndGet-4               2000000               782 ns/op               0 B/op          0 allocs/op
 
Benchmark_Cache_Struct_Set-4                     1000000              2766 ns/op             761 B/op          5 allocs/op
Benchmark_Cache_Struct_Set_Parallel-4            1000000              3243 ns/op             761 B/op          5 allocs/op
Benchmark_Cache_Struct_Get-4                     1000000              1925 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_Get_Parallel-4            1000000              2345 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_SetAndGet-4                300000              5638 ns/op            1078 B/op          7 allocs/op
```

размер кэша 1024&times;1024&times;10 - 16cpu
```
Benchmark_Cache_Circle_Set-16                    1000000              1207 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_Set_Parallel-16           1000000              1335 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_Get-16                    3000000               522 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_Get_Parallel-16          10000000               287 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Circle_SetAndGet-16              1000000              1277 ns/op               0 B/op          0 allocs/op
 
Benchmark_Cache_Struct_Set-16                    1000000              4124 ns/op             793 B/op          5 allocs/op
Benchmark_Cache_Struct_Set_Parallel-16            300000              4828 ns/op             774 B/op          5 allocs/op
Benchmark_Cache_Struct_Get-16                     500000              3491 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_Get_Parallel-16            300000              3804 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_SetAndGet-16               100000             27390 ns/op            1089 B/op          7 allocs/op
```