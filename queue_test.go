package main

import (
	"testing"

	git "github.com/libgit2/git2go/v28"
)

func TestQueue(t *testing.T) {
	c := &git.Commit{}
	q := newQueue()
	for i := 0; i < 4; i++ {
		q.push(c)
		q.push(c)
		q.push(c)
		if q.size() != i+3 {
			t.Fatalf("wrong size")
		}
		q.pop()
		q.pop()
		q.push(c)
		if q.size() != i+2 {
			t.Fatalf("wrong size")
		}
		q.pop()
		if q.size() != i+1 {
			t.Fatalf("wrong size")
		}
	}
}
