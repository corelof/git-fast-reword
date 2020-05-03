package main

import (
	"fmt"

	git "github.com/libgit2/git2go/v28"
)

type commit struct {
	parents      []*commit
	children     []*commit
	needsRebuild bool
	message      string
	id           string
}

type repoGraph struct {
	branchHeads  []*commit
	detachedHead bool
}

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
		if c == nil {
			return
		}
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

func buildCommitSubgraph(repoRoot string, neededCommits []string) (*repoGraph, error) {
	repo, err := git.OpenRepository(repoRoot)
	if err != nil {
		return nil, err
	}
	it, err := repo.NewBranchIterator(git.BranchLocal)
	if err != nil {
		return nil, err
	}
	topCommits := make([]*git.Commit, 0)
	if err = it.ForEach(func(b *git.Branch, t git.BranchType) error {
		if t != git.BranchLocal {
			return fmt.Errorf("wrong branch type")
		}
		cm, err := repo.LookupCommit(b.Target())
		if err != nil {
			return err
		}
		topCommits = append(topCommits, cm)
		return nil
	}); err != nil {
		return nil, err
	}
	detached, err := repo.IsHeadDetached()
	if err != nil {
		return nil, err
	}
	if detached {
		head, err := repo.Head()
		if err != nil {
			return nil, err
		}
		cm, err := repo.LookupCommit(head.Target())
		inTops := false
		for _, v := range topCommits {
			if v.Id() == cm.Id() {
				inTops = true
			}
		}
		if !inTops {
			topCommits = append(topCommits, cm)
		}
	}

	res := &repoGraph{branchHeads: make([]*commit, 0), detachedHead: detached}
	commits := make(map[string]*commit)
	q := newQueue()

	needed := make(map[string]bool)
	dsu := newDsu()

	for _, v := range neededCommits {
		needed[v] = true
		dsu.addNode(v)
	}

	for _, cm := range topCommits {
		q.push(cm)
		dsu.addNode(cm.Id().String())
	}

	parents := make(map[string][]string)
	children := make(map[string][]string)

	for q.size() > 0 {
		c := q.front()
		q.pop()
		dsu.addNode(c.Id().String())
		com, ok := commits[c.Id().String()]
		if ok {
			continue
		}
		com = &commit{
			message:      c.Message(),
			parents:      make([]*commit, 0),
			children:     make([]*commit, 0),
			id:           c.Id().String(),
			needsRebuild: true,
		}
		commits[com.id] = com
		n := c.ParentCount()
		var i uint
		if dsu.comp == 1 {
			break
		}
		for i = 0; i < n; i++ {
			q.push(c.Parent(i))
			if parents[com.id] == nil {
				parents[com.id] = make([]string, 0)
			}
			parents[com.id] = append(parents[com.id], c.Parent(i).Id().String())
			if children[c.Parent(i).Id().String()] == nil {
				children[c.Parent(i).Id().String()] = make([]string, 0)
			}
			dsu.connect(com.id, c.Parent(i).Id().String())
			children[c.Parent(i).Id().String()] = append(children[c.Parent(i).Id().String()], com.id)
		}
	}

	leafs := make([]*commit, 0)

	for _, v := range commits {
		for _, vv := range parents[v.id] {
			if commits[vv] != nil {
				v.parents = append(v.parents, commits[vv])
			}
		}
		for _, vv := range children[v.id] {
			v.children = append(v.children, commits[vv])
		}
		if len(v.parents) == 0 {
			leafs = append(leafs, v)
		}
	}

	u := make(map[string]bool)
	del := make(map[string]bool)
	for _, v := range neededCommits {
		needed[v] = true
	}
	deleteParent := func(c, p *commit) {
		idx := -1
		for i, v := range c.parents {
			if v.id == p.id {
				idx = i
			}
		}
		if idx != -1 {
			c.parents = append(c.parents[:idx], c.parents[idx+1:]...)
		}
		del[p.id] = true
	}

	pp := make(map[*commit]int)
	for _, v := range commits {
		pp[v] = len(v.parents)
	}

	var dfs func(*commit)
	dfs = func(c *commit) {
		if needed[c.id] {
			return
		}
		del[c.id] = true
		if u[c.id] {
			return
		}
		u[c.id] = true
		for _, v := range c.children {
			if needed[v.id] {
				deleteParent(v, c)
			} else {
				pp[v]--
				if pp[v] == 0 {
					dfs(v)
				}
			}
		}
	}

	for _, v := range leafs {
		dfs(v)
	}

	nc := make(map[string]*commit)

	for _, v := range commits {
		if !del[v.id] {
			nc[v.id] = v
		}
	}
	commits = nc

	for _, v := range commits {
		if len(v.parents) == 0 {
			leafs = append(leafs, v)
		}
	}

	for _, v := range leafs {
		oid, err := git.NewOid(v.id)
		if err != nil {
			return nil, err
		}
		cm, err := repo.LookupCommit(oid)
		if err != nil {
			return nil, err
		}
		n := cm.ParentCount()
		for i := 0; uint(i) < n; i++ {
			commit := &commit{
				needsRebuild: false,
				parents:      make([]*commit, 0),
				children:     []*commit{v},
				message:      cm.Parent(uint(i)).Message(),
				id:           cm.Parent(uint(i)).Id().String(),
			}
			commits[commit.id] = commit
			v.parents = append(v.parents, commit)
		}
	}

	for _, cm := range topCommits {
		if _, ok := commits[cm.Id().String()]; ok {
			res.branchHeads = append(res.branchHeads, commits[cm.Id().String()])
		}
	}

	return res, nil
}
