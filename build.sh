#! /bin/sh

. ./setup.sh

# build msg
cd src/rpc
./gen.sh
cd -

go test rpc || exit 1
cd src/rpc
go test -c
GODEBUG=gctrace=1
GODEBUG=schedtrace=1000
go test -bench=. -benchtime 10s  #-cpu 4 -cpuprofile cpu.out -memprofile mem.out -blockprofile block.out
cd -


