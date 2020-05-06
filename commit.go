package main

import (
	"fmt"
	"strconv"
	"strings"

	git "github.com/libgit2/git2go/v28"
)

// Converts relative 'path'(e.g. HEAD~2) to commit hash
func getCommitHash(repo *git.Repository, commit string) (string, error) {
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}
	headHash := ref.Branch().Target()

	if commit == "HEAD" {
		return headHash.String(), nil
	}

	cm, err := repo.LookupCommit(headHash)
	if err != nil {
		return "", err
	}

	type headRel struct {
		s byte
		n int
	}
	p := make([]headRel, 0)
	for i := 4; i < len(commit); {
		it := headRel{s: commit[i]}
		i++
		fp := i
		for i < len(commit) && commit[i] != '~' && commit[i] != '^' {
			i++
		}
		if fp == i {
			it.n = 1
		} else {
			it.n, err = strconv.Atoi(commit[fp:i])
			if err != nil {
				return "", err
			}
		}
		p = append(p, it)
	}
	for _, v := range p {
		if cm == nil {
			return "", fmt.Errorf("not found")
		}
		if v.s == '^' {
			cm = cm.Parent(uint(v.n) - 1)
		} else {
			for i := 0; i < v.n; i++ {
				cm = cm.Parent(0)
			}
		}
	}
	return cm.Id().String(), nil
}

func parseCommit(repo *git.Repository, commit string) (string, error) {
	var err error
	if strings.Contains(commit, "HEAD") {
		commit, err = getCommitHash(repo, commit)
		if err != nil {
			return "", err
		}
	}
	return commit, nil
}
