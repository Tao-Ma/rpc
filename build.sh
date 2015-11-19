#! /bin/sh

tmp='./tmp'

. ./setup.sh

# build msg
cd src/rpc
./gen.sh
cd -

if [ ! -e "$tmp" ]; then
	mkdir "$tmp" || (echo "ERROR: cannot create tmp direcotry" >&2 && exit 1)
fi

pkgname='rpc'

bin="$tmp/$pkgname.test"
cpuprofile="$tmp/cpu.out"
memprofile="$tmp/mem.out"
blockprofile="$tmp/block.out"
cpupng="$tmp/cpu.png"
memspng="$tmp/mems.png"
memopng="$tmp/memo.png"
blockpng="$tmp/block.png"

echo "INFO: build test binary file: $bin"
go test -o "$bin" "$pkgname" || exit 1

#GODEBUG=gctrace=1
#GODEBUG=schedtrace=1000
go test "$pkgname" -bench=. -benchtime 10s  #-cpu 4 -cpuprofile "$cpuprofile" -memprofile "$memprofile" -blockprofile "$blockprofile"

if [ "$gen_analyze" = "true" ]; then
	echo "png > $cpupng"  | go tool pprof -ignore=testing "$bin" "$cpuprofile"
	echo "png > $memopng" | go tool pprof -ignore=testing -alloc_objects "$bin" "$memprofile"
	echo "png > $memspng" | go tool pprof -ignore=testing -alloc_space "$bin" "$memprofile"
	echo "png > $blockpng"| go tool pprof -ignore=testing "$bin" "$blockpng"
fi
