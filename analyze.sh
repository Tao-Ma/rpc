#!/bin/sh

d=src/rpc
b=$d/rpc.test

echo 'png > cpu.png' | go tool pprof -ignore=testing $b $d/cpu.out
echo 'png > memo.png'| go tool pprof -ignore=testing -alloc_objects $b $d/mem.out
echo 'png > mems.png'| go tool pprof -ignore=testing -alloc_space $b $d/mem.out
echo 'png > block.png'| go tool pprof -ignore=testing $b $d/block.out
