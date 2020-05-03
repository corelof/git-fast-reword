package main

import (
	"container/list"

	git "github.com/libgit2/git2go/v28"
)

type queue struct {
	l *list.List
}

func newQueue() *queue {
	return &queue{list.New()}
}

func (q *queue) push(c *git.Commit) {
	q.l.PushBack(c)
}

func (q *queue) pop() {
	q.l.Remove(q.l.Front())
}

func (q *queue) size() int {
	return q.l.Len()
}

func (q *queue) front() *git.Commit {
	return q.l.Front().Value.(*git.Commit)
}
