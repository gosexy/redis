# Benchmarks

Here are some benchmarks for different Go redis clients.

```
go test -test.bench='.*'
PASS
BenchmarkAlphazeroRedisPing         50000      39001 ns/op
BenchmarkAlphazeroRedisSet          50000      43967 ns/op
BenchmarkAlphazeroRedisGet          50000      43459 ns/op
BenchmarkAlphazeroRedisIncr         50000       43350 ns/op
BenchmarkAlphazeroRedisLPush        50000       44249 ns/op
BenchmarkAlphazeroRedisLRange10     50000       58078 ns/op
BenchmarkAlphazeroRedisLRange100    10000      139603 ns/op
BenchmarkGaryburdRedigoPing         50000       51728 ns/op
BenchmarkGaryburdRedigoSet          50000       39938 ns/op
BenchmarkGaryburdRedigoGet          50000       36364 ns/op
BenchmarkGaryburdRedigoIncr         50000       53008 ns/op
BenchmarkGaryburdRedigoLPush        50000       41001 ns/op
BenchmarkGaryburdRedigoLRange10     50000       54167 ns/op
BenchmarkGaryburdRedigoLRange100    10000      132195 ns/op
BenchmarkGosexyRedisPing            50000       40768 ns/op
BenchmarkGosexyRedisSet             50000       44605 ns/op
BenchmarkGosexyRedisGet             50000       41317 ns/op
BenchmarkGosexyRedisIncr            50000       41673 ns/op
BenchmarkGosexyRedisLPush           50000       44169 ns/op
BenchmarkGosexyRedisLRange10        50000       59429 ns/op
BenchmarkGosexyRedisLRange100       10000      148187 ns/op
BenchmarkSimonz05GodisPing          50000       37014 ns/op
BenchmarkSimonz05GodisSet           50000       44739 ns/op
BenchmarkSimonz05GodisGet           50000       40958 ns/op
BenchmarkSimonz05GodisIncr          50000       42611 ns/op
BenchmarkSimonz05GodisLPush         50000       46353 ns/op
BenchmarkSimonz05GodisLRange10      50000       58262 ns/op
BenchmarkSimonz05GodisLRange100     10000      157230 ns/op
BenchmarkTcglRedisPing              10000      103713 ns/op
BenchmarkTcglRedisSet               10000      161212 ns/op
BenchmarkTcglRedisGet               10000      133768 ns/op
BenchmarkTcglRedisIncr              10000      133342 ns/op
BenchmarkTcglRedisLPush             10000      161915 ns/op
BenchmarkTcglRedisLRange10          10000      201212 ns/op
BenchmarkTcglRedisLRange100         10000      249613 ns/op
ok    menteslibres.net/gosexy/redis/_benchmarks 83.130s
```
