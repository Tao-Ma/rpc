// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package benchmark

import (
	"testing"
)

func TestCollector(t *testing.T) {
	c := NewCollecter(100)

	c.Add(2, 2, 3, 4, 4, 5, 6, 21, 37, 47, 100, 101)

	if c.Max() != 100 {
		t.Log(c.Max())
		t.Fail()
	}
	if c.Min() != 2 {
		t.Log(c.Min())
		t.Fail()
	}
	if c.Count() != 11 {
		t.Log(c.Count())
		t.Fail()
	}
	if c.Sum() != 231 {
		t.Log(c.Sum())
		t.Fail()
	}
	if c.Mean() != 21 {
		t.Log(c.Mean())
		t.Fail()
	}
	if m, n := c.OutOfRange(); n != 1 || m != 101 {
		t.Log(c.OutOfRange())
		t.Fail()
	}
	if c.Percentile(30.0) != 3 {
		t.Log(c.Percentile(30.0))
		t.Fail()
	}
	if c.Percentile(50.0) != 4 {
		t.Log(c.Percentile(50.0))
		t.Fail()
	}
	if c.Percentile(90.0) != 37 {
		t.Log(c.Percentile(90.0))
		t.Fail()
	}
}
