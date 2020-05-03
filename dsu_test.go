package main

import "testing"

func TestDsu(t *testing.T) {
	d := newDsu()
	checkSize := func(e int) {
		if d.comp != e {
			t.Errorf("expected size %d, found %d", e, d.comp)
		}
	}
	checkConnected := func(a, b string, e bool) {
		if d.connected(a, b) != e {
			t.Errorf("expected connection %s and %s is %v, found %v", a, b, e, d.connected(a, b))
		}
	}
	d.addNode("a")
	d.addNode("b")
	d.addNode("c")
	checkSize(3)
	d.connect("a", "b")
	checkSize(2)
	checkConnected("a", "b", true)
	checkConnected("a", "c", false)
	checkConnected("b", "c", false)
	checkConnected("c", "c", true)
	d.connect("c", "b")
	checkSize(1)
}
