package main

import (
	"fmt"

	git "github.com/libgit2/git2go/v28"
)

func buildCommitGraph(repoRoot string) (*repoGraph, error) {
	repo, err := git.OpenRepository(repoRoot)
	if err != nil {
		return nil, err
	}
	it, err := repo.NewBranchIterator(git.BranchLocal)
	if err != nil {
		return nil, err
	}
	bs := make([]*git.Branch, 0)
	if err = it.ForEach(func(b *git.Branch, t git.BranchType) error {
		if t != git.BranchLocal {
			return fmt.Errorf("wrong branch type")
		}
		bs = append(bs, b)
		return nil
	}); err != nil {
		return nil, err
	}

	res := &repoGraph{branchHeads: make([]*commit, 0)}
	commits := make(map[string]*commit)

	var dfs func(*git.Commit) error
	dfs = func(c *git.Commit) error {
		if c == nil {
			return fmt.Errorf("nil commit received")
		}
		com, ok := commits[c.Id().String()]
		if ok {
			return nil
		}
		com = &commit{
			message:  c.Message(),
			parents:  make([]*commit, 0),
			children: make([]*commit, 0),
			id:       c.Id().String(),
		}
		commits[c.Id().String()] = com
		n := c.ParentCount()
		var i uint
		for i = 0; i < n; i++ {
			if err = dfs(c.Parent(i)); err != nil {
				return err
			}
			com.parents = append(com.parents, commits[c.ParentId(i).String()])
			commits[c.ParentId(i).String()].children = append(commits[c.ParentId(i).String()].children, com)
		}
		return nil
	}

	for _, b := range bs {
		cm, err := repo.LookupCommit(b.Target())
		if err != nil {
			return nil, err
		}
		if err := dfs(cm); err != nil {
			return nil, err
		}
		cm, err = repo.LookupCommit(b.Target())
		if err != nil {
			return nil, err
		}
		res.branchHeads = append(res.branchHeads, commits[cm.Id().String()])
	}
	return res, nil
}

type commit struct {
	parents  []*commit
	children []*commit
	message  string
	id       string
}

type repoGraph struct {
	branchHeads []*commit
}

// TODO optimize buildGraph. It should work only with subgraph, containing all affected commits, we can build it with bfs
// But it can break current iteractiveReword implementation

func (g *repoGraph) Reword(params []rewordParam) {
	newMessage := make(map[string]string)
	for _, v := range params {
		newMessage[v.hash] = v.message
	}
	var dfs func(*commit)
	dfs = func(c *commit) {
		if c == nil {
			return
		}
		nm, ok := newMessage[c.id]
		if ok {
			c.message = nm
		}
		for _, v := range c.parents {
			dfs(v)
		}
	}
	for _, v := range g.branchHeads {
		dfs(v)
	}
}

func (g *repoGraph) TopSort() []*commit {
	res := make([]*commit, 0)
	u := make(map[*commit]bool)
	var dfs func(*commit)
	dfs = func(c *commit) {
		u[c] = true
		for _, next := range c.parents {
			if !u[next] {
				dfs(next)
			}
		}
		res = append(res, c)
	}
	for _, s := range g.branchHeads {
		if !u[s] {
			dfs(s)
		}
	}
	return res
}
