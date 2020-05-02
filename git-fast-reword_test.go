package main

import (
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"
)

const (
	testDir        = "test_repos"
	interactiveDir = testDir + "/interactive"
	fastDir        = testDir + "/fast"

	newCommitMessage   = "new_commit"
	initRepoScriptsDir = "init_repo"
	branchName         = "Fast_Reword_Branch"
	testCases          = 4
	outputHashSize     = 40
)

func interactiveRebaseReword(repoRoot string, commitHash string, newMessage string) error {
	// TODO
	return nil
}

func bruteReword(repoRoot string, params []rewordParam) error {
	for _, v := range params {
		if err := interactiveRebaseReword(repoRoot, v.hash, v.message); err != nil {
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

			if err := bruteReword(interactiveDir, params1); err != nil {
				t.Error(err)
			}
			if err := fastReword(fastDir, params2); err != nil {
				t.Error(err)
			}

			compareRepos(t, interactiveDir, fastDir)
		})
		os.RemoveAll(testDir)
	}
}

func compareRepos(t *testing.T, repo1, repo2 string) {
	g1, err := buildCommitGraph(repo1)
	if err != nil {
		t.Error(err)
	}
	g2, err := buildCommitGraph(repo1)
	if err != nil {
		t.Error(err)
	}
	if !g1.Equal(g2) {
		t.Fatalf("repo graphs are not equal")
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
