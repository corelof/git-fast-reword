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
	_ = err
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
		if !ok {
			com = &commit{
				message:  c.Message(),
				parents:  make([]*commit, 0),
				children: make([]*commit, 0),
				id:       c.Id().String(),
			}
			commits[c.Id().String()] = com
		}
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
