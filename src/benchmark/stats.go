// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package benchmark

import ()

type Collector struct {
	max uint64
	min uint64
	num uint64

	// out of range data
	oor_num uint64
	oor_max uint64

	dlen uint64
	data []uint64
}

func NewCollecter(max uint64) *Collector {
	c := new(Collector)

	// include "0"
	c.dlen = max + 1
	c.data = make([]uint64, c.dlen)

	c.Reset()

	return c
}

func (c *Collector) add(d uint64) {
	if d >= c.dlen {
		c.oor_num++
		if d > c.oor_max {
			c.oor_max = d
		}
		return
	}

	c.num++
	c.data[d]++
	if d > c.max {
		c.max = d
	}
	if d < c.min {
		c.min = d
	}
}

func (c *Collector) Add(stat ...uint64) {
	for _, s := range stat {
		c.add(s)
	}
}

func (c *Collector) Reset() {
	for i := c.min; i <= c.max; i++ {
		c.data[i] = 0
	}

	c.max = 0
	c.min = c.dlen
	c.num = 0

	c.oor_num = 0
	c.oor_max = 0
}

func (c *Collector) Do(f func(uint64, uint64)) {
	for i := c.min; i <= c.max; i++ {
		f(i, c.data[i])
	}
}

func (c *Collector) Sum() uint64 {
	var sum uint64 = 0

	c.Do(func(v uint64, n uint64) { sum += v * n })

	return sum
}

func (c *Collector) Count() uint64 {
	return c.num
}

func (c *Collector) Mean() float64 {
	sum := c.Sum()
	count := c.Count()

	return float64(sum) / float64(count)
}

func (c *Collector) Max() uint64 {
	return c.max
}

func (c *Collector) Min() uint64 {
	return c.min
}

func (c *Collector) OutOfRange() (uint64, uint64) {
	return c.oor_max, c.oor_num
}

func (c *Collector) Percentile(p float64) uint64 {
	total := uint64((p / 100.0) * float64(c.Count()))
	num := uint64(0)
	val := uint64(0)

	c.Do(func(v uint64, n uint64) {
		if num >= total {
			return
		}
		val = v
		num += n
	})

	return val
}

func (c *Collector) Stddev() float64 {
	return 0.0
}

func (c *Collector) WithinStddev() float64 {
	return 0.0
}
