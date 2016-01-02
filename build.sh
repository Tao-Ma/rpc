#! /bin/sh

tmp='./tmp'

. ./setup.sh

# build msg
cd src/rpc
./msg_gen.sh
cd -

if [ ! -e "$tmp" ]; then
	mkdir "$tmp" || (echo "ERROR: cannot create tmp direcotry" >&2 && exit 1)
fi

pkgname='rpc'
profile=true

bin="$tmp/$pkgname.test"
cpuprofile="$tmp/cpu.out"
memprofile="$tmp/mem.out"
blockprofile="$tmp/block.out"
cpupng="$tmp/cpu.png"
memspng="$tmp/mems.png"
memopng="$tmp/memo.png"
blockpng="$tmp/block.png"

echo "INFO: build test binary file: $bin"

#export GODEBUG=gctrace=1
#export GODEBUG=schedtrace=1000
if [ "$profile" = "true" ]; then
	profile_flags="-o $bin -cpuprofile $cpuprofile -memprofile $memprofile -blockprofile $blockprofile"
fi

go test "$pkgname" -bench=Router -timeout 10m -benchtime 20s -cpu 8 $profile_flags

if [ "$profile" = "true" ]; then
	echo "png > $cpupng"  | go tool pprof -ignore=testing "$bin" "$cpuprofile"
	echo "png > $memopng" | go tool pprof -ignore=testing -alloc_objects "$bin" "$memprofile"
	echo "png > $memspng" | go tool pprof -ignore=testing -alloc_space "$bin" "$memprofile"
	echo "png > $blockpng"| go tool pprof -ignore=testing "$bin" "$blockprofile"
fi
