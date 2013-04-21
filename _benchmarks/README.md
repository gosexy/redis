# Benchmarks

Here are some benchmarks for different Go redis clients.

Benchmarks should not be taken as ultimate proofs of how a program or library
behaves as they may throw mixed results across different hosts.

You can use the available benchmarking tools to reproduce these results in your
own host.

Please see [getong's blog][1] for a related blog post.

## Benchmarking graphs

Published benchmarks results use latest available version (as of Apr 20, 2013)
of all the tested libraries.

Tests were against a redis 2.6.12 server with these settings modified in order
to keep the noise at the minimum:

```
# redis.conf
loglevel warning
save ""
```

Each colored line represent a library executing one function
(PING, SET, GET, INCR, LPUSH, LRANGE) and reading a result from the redis
server.

The height of the line represent the average time in nanoseconds the function
took to execute.

Exhibit A: Debian virtual machine 256Mb ram, 1 core 2GHz (qemu).

Test #1

<img src="results/20130420/cpu1-mem512/redis-benchmarks-1.png" width="800" />

Test #2

<img src="results/20130420/cpu1-mem512/redis-benchmarks-2.png" width="200" />

Test #3

<img src="results/20130420/cpu1-mem512/redis-benchmarks-3.png" width="200" />

Exhibit B: OSX host 4G ram, 4 cores 2.3GHz.

Test #1

<img src="results/20130420/cpu4-mem4/redis-benchmarks-1.png" width="800" />

Test #2

<img src="results/20130420/cpu4-mem4/redis-benchmarks-2.png" width="200" />

Test #3

<img src="results/20130420/cpu4-mem4/redis-benchmarks-3.png" width="200" />

Exhibit C: Debian virtual machine 16G ram, 8 cores 2GHz (qemu).

Test #1
<img src="results/20130420/cpu8-mem16/redis-benchmarks-1.png" width="800" />

Test #2
<img src="results/20130420/cpu8-mem16/redis-benchmarks-2.png" width="200" />

Test #3
<img src="results/20130420/cpu8-mem16/redis-benchmarks-3.png" width="200" />

## How to reproduce benchmarking

In order to generate the results plot you'll need the [R][2] programming
language interpreter.

```
# The R programming language
aptitude install r-core
# Version control systems wrapped by go get.
aptitude install mercurial git bzr
```

Then update all the related libraries to their latest version.

```
go get -u code.google.com/p/tcgl/redis
go get -u github.com/alphazero/Go-Redis
go get -u github.com/garyburd/redigo/redis
go get -u github.com/simonz05/godis/redis
go get -u menteslibres.net/gosexy/redis
cd $GOPATH/src/menteslibres.net/gosexy/redis/_benchmarks
```

Finally, use the `./bench.sh` file to run a benchmarking test and generate a
plot.

```
$ ./bench.sh
$ file redis-benchmarks.png
redis-benchmarks.png: PNG image data, 1400 x 900, 8-bit/color RGB, non-interlaced
```

[1]: http://www.cnblogs.com/getong/archive/2013/04/01/2993139.html
[2]: http://www.r-project.org/
