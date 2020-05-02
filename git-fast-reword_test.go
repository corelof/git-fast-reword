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
	testDir        = "test_repos"
	interactiveDir = testDir + "/interactive"
	fastDir        = testDir + "/fast"

	newCommitMessage   = "new_commit\n"
	initRepoScriptsDir = "init_repo"
	branchName         = "Fast_Reword_Branch"
	testCases          = 1
	outputHashSize     = 40
)

// Works only for chains
func interactiveRebaseReword(repoRoot string, params []rewordParam) error {
	g, err := buildCommitGraph(repoRoot)
	if err != nil {
		return err
	}
	rebaseOnRoot := false
	commits := make([]*commit, 0)
	for _, v := range params {
		c := g.GetCommit(v.hash)
		if c == nil {
			return fmt.Errorf("commit not found")
		}
		if len(c.parents) > 1 {
			return fmt.Errorf("branching / merges not supported")
		}
		if len(c.parents) == 0 {
			rebaseOnRoot = true
		}
		commits = append(commits, c)
	}
	cnf := make(map[string]bool)
	for _, v := range commits {
		cnf[v.id] = true
	}
	repo, err := git.OpenRepository(repoRoot)
	if err != nil {
		return err
	}
	hr, err := repo.Head()
	if err != nil {
		return err
	}
	topcommit, err := repo.LookupCommit(hr.Branch().Target())
	if err != nil {
		return err
	}
	tc := g.GetCommit(topcommit.Id().String())
	var last string
	commitOrder := make([]string, 0)
	for {
		commitOrder = append(commitOrder, tc.id)
		if cnf[tc.id] {
			delete(cnf, tc.id)
		}
		if len(cnf) == 0 {
			last = tc.id
			break
		}
		if len(tc.parents) != 1 {
			return fmt.Errorf("branching / merges not supported or HEAD is not on the same branch as one of commits")
		}
		tc = tc.parents[0]
	}
	c := g.GetCommit(last)
	if !rebaseOnRoot {
		c = c.parents[0]
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	what := c.id
	if rebaseOnRoot {
		what = "--root"
	}

	newMessage := make(map[string]string)
	for _, v := range params {
		newMessage[v.hash] = v.message
	}

	var rebaseConfig string
	mm := make([]string, 0)
	for idx := len(commitOrder) - 1; idx >= 0; idx-- {
		if newmsg, ok := newMessage[commitOrder[idx]]; ok {
			rebaseConfig += fmt.Sprintf("e %s\n", commitOrder[idx])
			mm = append(mm, newmsg)
		} else {
			rebaseConfig += fmt.Sprintf("p %s\n", commitOrder[idx])

		}
	}
	cmd := fmt.Sprintf("cd $(pwd)/%s && GIT_SEQUENCE_EDITOR=\"%s/configure_interactive_rebase.sh '%s'\" git rebase -i %s", repoRoot, wd, rebaseConfig, what)
	if err = exec.Command("/bin/sh", "-c", cmd).Run(); err != nil {
		return err
	}

	for _, v := range mm {
		cmd = fmt.Sprintf("cd $(pwd)/%s && git commit --amend -m \"%s\" && git rebase --continue", repoRoot, v)
		if err = exec.Command("/bin/sh", "-c", cmd).Run(); err != nil {
			return err
		}
	}
	return nil
}

func TestMain(t *testing.T) {
	for i := 1; i <= testCases; i++ {
		t.Run("test "+strconv.Itoa(i), func(t *testing.T) {
			hashes1, err := exec.Command("./"+initRepoScriptsDir+"/"+strconv.Itoa(i)+".sh", interactiveDir).Output()
			if err != nil {
				t.Error(err)
			}
			hashes2, err := exec.Command("./"+initRepoScriptsDir+"/"+strconv.Itoa(i)+".sh", fastDir).Output()
			if err != nil {
				t.Error(err)
			}

			params1 := make([]rewordParam, 0)
			for i := 0; i < len(hashes1)/outputHashSize; i++ {
				params1 = append(params1,
					rewordParam{string(hashes1[i*outputHashSize : (i+1)*outputHashSize]), newCommitMessage},
				)
			}
			params2 := make([]rewordParam, 0)
			for i := 0; i < len(hashes2)/outputHashSize; i++ {
				params2 = append(params2,
					rewordParam{string(hashes2[i*outputHashSize : (i+1)*outputHashSize]), newCommitMessage},
				)
			}

			g, err := buildCommitGraph(interactiveDir)
			if err != nil {
				t.Error(err)
			}
			g.Reword(params1)

			if err := interactiveRebaseReword(interactiveDir, params1); err != nil {
				t.Error(err)
			}
			if err := fastReword(fastDir, params2); err != nil {
				t.Error(err)
			}

			compareRepos(t, interactiveDir, fastDir, g)
		})
		os.RemoveAll(testDir)
	}
}

func compareRepos(t *testing.T, repo1, repo2 string, g *repoGraph) {
	g1, err := buildCommitGraph(repo1)
	if err != nil {
		t.Error(err)
	}
	g2, err := buildCommitGraph(repo2)
	if err != nil {
		t.Error(err)
	}
	if !g1.Equal(g) {
		t.Errorf("interactive rebased graph is wrong")
	}
	return
	// TODO
	if !g2.Equal(g) {
		t.Errorf("fast reworded graph is wrong")
	}
}

func (g *repoGraph) Equal(g2 *repoGraph) bool {
	if len(g.branchHeads) != len(g2.branchHeads) {
		return false
	}

	type cItem struct {
		m  string
		pm []string
	}
	var m1, m2 = make([]cItem, 0), make([]cItem, 0)
	var dfs func(*commit, bool)
	dfs = func(c *commit, f bool) {
		if c == nil {
			return
		}
		cur := cItem{m: c.message, pm: make([]string, 0)}
		for _, v := range c.parents {
			dfs(v, f)
			cur.pm = append(cur.pm, v.message)
		}
		if f {
			m1 = append(m1, cur)
		} else {
			m2 = append(m2, cur)
		}
	}
	for _, v := range g.branchHeads {
		dfs(v, true)
	}
	for _, v := range g2.branchHeads {
		dfs(v, false)
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
