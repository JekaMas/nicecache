размер кэша 1024&times;1024
```
Benchmark_Cache_Nice_Set-4                     3000000               391 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-4            2000000               805 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-4                     5000000               404 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-4           10000000               139 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-4               3000000               508 ns/op               0 B/op          0 allocs/op

     
Benchmark_Cache_Struct_Set-4                     2000000               894 ns/op             219 B/op          1 allocs/op
Benchmark_Cache_Struct_Set_Parallel-4            1000000              1151 ns/op             247 B/op          1 allocs/op
Benchmark_Cache_Struct_Get-4                     1000000              1952 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_Get_Parallel-4            1000000              2732 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_SetAndGet-4               1000000              3247 ns/op             583 B/op          3 allocs/op
```

размер кэша 1024&times;1024&times;10
```
Benchmark_Cache_Nice_Set-4                     3000000               472 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-4            2000000               715 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-4                     3000000               416 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-4           10000000               167 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-4               2000000               588 ns/op               0 B/op          0 allocs/op
 
Benchmark_Cache_Struct_Set-4                     1000000              2766 ns/op             761 B/op          5 allocs/op
Benchmark_Cache_Struct_Set_Parallel-4            1000000              3243 ns/op             761 B/op          5 allocs/op
Benchmark_Cache_Struct_Get-4                     1000000              1925 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_Get_Parallel-4            1000000              2345 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_SetAndGet-4                300000              5638 ns/op            1078 B/op          7 allocs/op
```

размер кэша 1024&times;1024&times;10 - 16cpu
```
Benchmark_Cache_Nice_Set-16                    3000000               488 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Set_Parallel-16           2000000               789 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get-16                    3000000               423 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_Get_Parallel-16          10000000               177 ns/op               0 B/op          0 allocs/op
Benchmark_Cache_Nice_SetAndGet-16              2000000               633 ns/op               0 B/op          0 allocs/op
 
Benchmark_Cache_Struct_Set-16                    1000000              4124 ns/op             793 B/op          5 allocs/op
Benchmark_Cache_Struct_Set_Parallel-16            300000              4828 ns/op             774 B/op          5 allocs/op
Benchmark_Cache_Struct_Get-16                     500000              3491 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_Get_Parallel-16            300000              3804 ns/op             336 B/op          2 allocs/op
Benchmark_Cache_Struct_SetAndGet-16               100000             27390 ns/op            1089 B/op          7 allocs/op
```