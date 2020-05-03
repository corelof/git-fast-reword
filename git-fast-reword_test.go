package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"

	git "github.com/libgit2/git2go/v28"
)

const (
	testDir = "test_repo/"

	newCommitMessage   = "new_commit\n"
	initRepoScriptsDir = "init_repo"
	outputHashLen      = 40
)

func TestMain(t *testing.T) {
	ird, err := os.Open(initRepoScriptsDir)
	if err != nil {
		t.Fatal(err)
	}
	names, err := ird.Readdirnames(0)
	if err != nil {
		t.Fatal(err)
	}
	_ = len(names)
	for i := 1; i <= len(names); i++ {
		t.Run("test "+strconv.Itoa(i), func(t *testing.T) {
			hashes, err := exec.Command("./"+initRepoScriptsDir+"/"+strconv.Itoa(i)+".sh", testDir).Output()
			if err != nil {
				t.Error(err)
			}
			params := make([]rewordParam, 0)
			for j := 0; j < len(hashes)/outputHashLen; j++ {
				params = append(params,
					rewordParam{string(hashes[j*outputHashLen : (j+1)*outputHashLen]), newCommitMessage},
				)
			}
			g, err := buildFullCommitGraph(testDir)
			if err != nil {
				t.Error(err)
			}
			g.Reword(params)
			if err := fastReword(testDir, params); err != nil {
				t.Error(err)
			}

			g1, err := buildFullCommitGraph(testDir)
			if err != nil {
				t.Error(err)
			}
			if !g1.Equal(g) {
				t.Errorf("fast rebased graph is wrong")
			}
		})
		os.RemoveAll(testDir)
	}
}

func (g *repoGraph) Equal(g2 *repoGraph) bool {
	if g.detachedHead != g2.detachedHead {
		return false
	}
	if len(g.branchHeads) != len(g2.branchHeads) {
		return false
	}

	type cItem struct {
		m  string
		pm []string
	}
	var m1, m2 = make([]cItem, 0), make([]cItem, 0)
	u := make(map[*commit]bool)
	var dfs func(*commit, bool)
	dfs = func(c *commit, f bool) {
		if c == nil {
			return
		}
		u[c] = true
		cur := cItem{m: c.message, pm: make([]string, 0)}
		for _, v := range c.parents {
			if !u[v] {
				dfs(v, f)
				cur.pm = append(cur.pm, v.message)
			}
		}
		if f {
			m1 = append(m1, cur)
		} else {
			m2 = append(m2, cur)
		}
	}
	for _, v := range g.branchHeads {
		if !u[v] {
			dfs(v, true)
		}
	}
	for _, v := range g2.branchHeads {
		if !u[v] {
			dfs(v, false)
		}
	}

	cItemEqual := func(i1 cItem, i2 cItem) bool {
		if i1.m != i2.m {
			return false
		}
		mm1, mm2 := make(map[string]uint), make(map[string]uint)
		for _, v := range i1.pm {
			mm1[v]++
		}
		for _, v := range i2.pm {
			mm2[v]++
		}
		return reflect.DeepEqual(mm1, mm2)
	}

	if len(m1) != len(m2) {
		return false
	}
	for _, v := range m1 {
		fidx := -1
		for idx, vv := range m2 {
			if cItemEqual(v, vv) {
				fidx = idx
				break
			}
		}
		if fidx == -1 {
			return false
		}
		m2 = append(m2[:fidx], m2[fidx+1:]...)
	}
	return len(m2) == 0
}

func (g *repoGraph) GetCommit(hash string) *commit {
	var dfs func(*commit) *commit
	dfs = func(c *commit) *commit {
		if c == nil {
			return nil
		}
		if c.id == hash {
			return c
		}
		for _, v := range c.parents {
			if res := dfs(v); res != nil {
				return res
			}
		}
		return nil
	}
	for _, v := range g.branchHeads {
		if res := dfs(v); res != nil {
			return res
		}
	}
	return nil
}

func buildFullCommitGraph(repoRoot string) (*repoGraph, error) {
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
		topCommits = append(topCommits, cm)
	}

	res := &repoGraph{branchHeads: make([]*commit, 0), detachedHead: detached}
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
			if err = dfs(c.Parent(i)); err != nil {
				return err
			}
			com.parents = append(com.parents, commits[c.ParentId(i).String()])
			commits[c.ParentId(i).String()].children = append(commits[c.ParentId(i).String()].children, com)
		}
		return nil
	}

	for _, cm := range topCommits {
		if err := dfs(cm); err != nil {
			return nil, err
		}
		res.branchHeads = append(res.branchHeads, commits[cm.Id().String()])
	}

	return res, nil
}
