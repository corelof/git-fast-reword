package main

import (
	"os"
	"os/exec"
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

		os.RemoveAll(testDir)
	}
}

func compareRepos(t *testing.T, repo1, repo2 string) {
	// TODO
}
