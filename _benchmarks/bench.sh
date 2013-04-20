#!/bin/sh
#
# Please see:
# http://www.cnblogs.com/getong/archive/2013/04/01/2993139.html
#

go test -test.benchtime=2 -test.bench=. > redis-go-driver-benchmark.tmp

for i in AlphazeroRedis GaryburdRedigo GosexyRedis Simonz05Godis TcglRedis
do
  grep $i redis-go-driver-benchmark.tmp  | awk '{print $3}' > $i.tmp
done

R --no-save < go-redis-getongs-data.R > output.log
rm -f *.tmp
