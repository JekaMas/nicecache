размер кэша 1024&times;1024
```
Benchmark_Cache_Nice_Set-4                      20000000               275 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-4             50000000               134 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-4                      10000000               522 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-4             20000000               339 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-4                20000000               433 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet_Parallel-4       10000000               640 ns/op               0 B/op          0 allocs/op
     
Benchmark_Cache_Struct_Set-4                     2000000               894 ns/op             219 B/op          1 allocs/op
Benchmark_Cache_Struct_Set_Parallel-4            1000000              1151 ns/op             247 B/op          1 allocs/op
Benchmark_Cache_Struct_Get-4                     1000000              1952 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_Get_Parallel-4            1000000              2732 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_SetAndGet-4               1000000              3247 ns/op             583 B/op          3 allocs/op
```

размер кэша 1024&times;1024&times;10
```
Benchmark_Cache_Nice_Set-4                      20000000               312 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-4             50000000               168 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-4                      10000000               616 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-4             20000000               463 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-4                20000000               449 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet_Parallel-4       10000000               698 ns/op               0 B/op          0 allocs/op

    
Benchmark_Cache_Struct_Set-4                     1000000              2766 ns/op             761 B/op          5 allocs/op
Benchmark_Cache_Struct_Set_Parallel-4            1000000              3243 ns/op             761 B/op          5 allocs/op
Benchmark_Cache_Struct_Get-4                     1000000              1925 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_Get_Parallel-4            1000000              2345 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_SetAndGet-4                300000              5638 ns/op            1078 B/op          7 allocs/op
```

размер кэша 1024&times;1024&times;10 - 16cpu
```
Benchmark_Cache_Nice_Set-16                     30000000               287 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-16            50000000               139 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-16                     10000000               648 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-16            30000000               374 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-16               20000000               516 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet_Parallel-16      10000000               875 ns/op               0 B/op          0 allocs/op

 
Benchmark_Cache_Struct_Set-16                    1000000              4124 ns/op             793 B/op          5 allocs/op
Benchmark_Cache_Struct_Set_Parallel-16            300000              4828 ns/op             774 B/op          5 allocs/op
Benchmark_Cache_Struct_Get-16                     500000              3491 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_Get_Parallel-16            300000              3804 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_SetAndGet-16               100000             27390 ns/op            1089 B/op          7 allocs/op
```