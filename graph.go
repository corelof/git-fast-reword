package main

import (
	"fmt"
	"time"

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

// Reword updates messages of some commits according to params
func (g *repoGraph) Reword(params []rewordParam) {
	newMessage := make(map[string]string)
	u := make(map[string]struct{})
	for _, v := range params {
		newMessage[v.hash] = v.message
	}
	var dfs func(*commit)
	dfs = func(c *commit) {
		u[c.id] = struct{}{}
		nm, ok := newMessage[c.id]
		if ok {
			c.message = nm
		}
		for _, v := range c.parents {
			if _, ok := u[v.id]; !ok {
				dfs(v)
			}
		}
	}
	for _, v := range g.branchHeads {
		if _, ok := u[v.id]; !ok {
			dfs(v)
		}
	}
}

// TopSort returns topological sort of commit graph
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

// returns local branch / tag / detached head targets, optimizes by date / headonly if required
func getTopCommits(repo *git.Repository, dateOptimization, headOnly bool, earliestRequiredCommit time.Time) ([]*git.Commit, error) {
	topCommits := make([]*git.Commit, 0)

	if headOnly {
		head, err := repo.Head()
		if err != nil {
			return nil, err
		}
		cm, err := repo.LookupCommit(head.Target())
		topCommits = append(topCommits, cm)
	} else {
		inTopCommits := make(map[string]struct{})
		it, err := repo.NewBranchIterator(git.BranchLocal)
		if err != nil {
			return nil, err
		}

		if err = it.ForEach(func(b *git.Branch, t git.BranchType) error {
			if t != git.BranchLocal {
				return fmt.Errorf("wrong branch type")
			}
			cm, err := repo.LookupCommit(b.Target())
			if err != nil {
				return err
			}
			if dateOptimization && cm.Committer().When.Before(earliestRequiredCommit) {
				return nil
			}
			if _, ok := inTopCommits[cm.Id().String()]; !ok {
				inTopCommits[cm.Id().String()] = struct{}{}
				topCommits = append(topCommits, cm)
			}
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
			if !(dateOptimization && cm.Committer().When.Before(earliestRequiredCommit)) {
				if _, ok := inTopCommits[cm.Id().String()]; !ok {
					inTopCommits[cm.Id().String()] = struct{}{}
					topCommits = append(topCommits, cm)
				}
			}
		}

		if err = repo.Tags.Foreach(func(name string, obj *git.Oid) error {
			ref, err := repo.References.Lookup(name)
			if err != nil {
				return err
			}
			lightweightTag := false
			t, err := repo.LookupTag(obj)
			if err != nil {
				lightweightTag = true
			}
			if lightweightTag {
				if _, ok := inTopCommits[ref.Target().String()]; !ok {
					inTopCommits[ref.Target().String()] = struct{}{}
					cm, err := repo.LookupCommit(ref.Target())
					if err != nil {
						return err
					}
					if dateOptimization && cm.Committer().When.Before(earliestRequiredCommit) {
						return nil
					}
					topCommits = append(topCommits, cm)
				}
			} else {
				cm, err := t.Target().AsCommit()
				if err != nil {
					return err
				}
				if dateOptimization && cm.Committer().When.Before(earliestRequiredCommit) {
					return nil
				}
				if _, ok := inTopCommits[cm.Id().String()]; !ok {
					inTopCommits[cm.Id().String()] = struct{}{}
					topCommits = append(topCommits, cm)
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return topCommits, nil
}

func buildCommitSubgraph(repo *git.Repository, neededCommits []string, dateOptimization, headOnly bool) (*repoGraph, error) {
	needed := make(map[string]bool)
	var earliest time.Time
	upd := false
	for _, v := range neededCommits {
		needed[v] = true
		oid, err := git.NewOid(v)
		if err != nil {
			return nil, err
		}
		cm, err := repo.LookupCommit(oid)
		dt := cm.Committer().When.UTC()
		if !upd || dt.Before(earliest) {
			earliest = dt
			upd = true
		}
	}

	topCommits, err := getTopCommits(repo, dateOptimization, headOnly, earliest)
	if err != nil {
		return nil, err
	}

	detached, err := repo.IsHeadDetached()
	if err != nil {
		return nil, err
	}

	res := &repoGraph{branchHeads: make([]*commit, 0), detachedHead: detached}
	commits := make(map[string]*commit)

	// Select some subgraph containing all required commits
	var dfs func(*git.Commit) error
	dfs = func(c *git.Commit) error {
		if c == nil {
			return fmt.Errorf("nil commit")
		}
		if dateOptimization && c.Committer().When.Before(earliest) && !needed[c.Id().String()] {
			return nil
		}
		com, ok := commits[c.Id().String()]
		if ok {
			return nil
		}
		com = &commit{
			message:      c.Message(),
			parents:      make([]*commit, 0),
			children:     make([]*commit, 0),
			id:           c.Id().String(),
			needsRebuild: true,
		}
		commits[c.Id().String()] = com
		n := c.ParentCount()
		var i uint
		for i = 0; i < n; i++ {
			if err := dfs(c.Parent(i)); err != nil {
				return err
			}
			if pc, ok := commits[c.ParentId(i).String()]; ok {
				com.parents = append(com.parents, pc)
				commits[pc.id].children = append(commits[pc.id].children, com)
			}
		}
		return nil
	}
	for _, v := range topCommits {
		if err := dfs(v); err != nil {
			return nil, err
		}
	}

	// Drop guaranteed useless commits
	leafs := make([]*commit, 0)
	for _, v := range commits {
		if len(v.parents) == 0 {
			leafs = append(leafs, v)
		}
	}
	u := make(map[string]bool)
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
		u[p.id] = true
	}

	// Create some commits as parents that we will not update(to use hashes as parents value)
	pp := make(map[string]int)
	for _, v := range commits {
		pp[v.id] = len(v.parents)
	}

	var dfs2 func(*commit)
	dfs2 = func(c *commit) {
		if needed[c.id] {
			return
		}
		if u[c.id] {
			return
		}
		u[c.id] = true
		for _, v := range c.children {
			if needed[v.id] {
				deleteParent(v, c)
			} else {
				pp[v.id]--
				if pp[v.id] == 0 {
					dfs2(v)
				} else {
					deleteParent(v, c)
				}
			}
		}
	}

	for _, v := range leafs {
		dfs2(v)
	}

	nc := make(map[string]*commit)
	for _, v := range commits {
		if !u[v.id] {
			nc[v.id] = v
		}
	}
	commits = nc

	tadd := make([]*commit, 0)

	for _, v := range commits {
		oid, err := git.NewOid(v.id)
		if err != nil {
			return nil, err
		}
		cm, err := repo.LookupCommit(oid)
		if err != nil {
			return nil, err
		}
		pc := cm.ParentCount()
		var i uint
		for i = 0; i < pc; i++ {
			c := cm.Parent(i)
			if _, ok := commits[c.Id().String()]; !ok {
				commit := &commit{
					needsRebuild: false,
					parents:      make([]*commit, 0),
					children:     []*commit{v},
					message:      c.Message(),
					id:           c.Id().String(),
				}
				tadd = append(tadd, commit)
				v.parents = append(v.parents, commit)
			}
		}
	}

	for _, v := range tadd {
		commits[v.id] = v
	}

	for _, cm := range topCommits {
		if _, ok := commits[cm.Id().String()]; ok {
			res.branchHeads = append(res.branchHeads, commits[cm.Id().String()])
		}
	}

	return res, nil
}
