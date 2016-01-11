package benchmark

import (
	"testing"
)

func TestCollector(t *testing.T) {
	c := NewCollecter(100)

	c.Add(2, 2, 3, 4, 4, 5, 6, 9, 21, 35, 100, 101)

	t.Log(c.Max())
	t.Log(c.Min())
	t.Log(c.Count())
	t.Log(c.Sum())
	t.Log(c.Mean())
	t.Log(c.OutOfRange())
	t.Log(c.Percentile(30.0))
	t.Log(c.Percentile(50.0))
	t.Log(c.Percentile(90.0))

	t.Fail()
}
