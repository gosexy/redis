# gosexy/redis

Go bindings for the official [Redis][1] client, [hiredis][2].

## How to install

The standard `go get` will not work as we are using submodules to fetch
the [hiredis][2] source code.

```
mkdir -p $GOPATH/src/github.com/gosexy/redis
cd $GOPATH/src/github.com/gosexy/redis
git clone https://github.com/gosexy/redis.git .
git submodule init
git submodule update
go build
```

[1]: http://redis.io
[2]: https://github.com/redis/hiredis
