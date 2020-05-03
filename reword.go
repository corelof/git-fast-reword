package main

import (
	"log"
	"os/exec"

	git "github.com/libgit2/git2go/v28"
)

// TODO add randomly generated repositories

func fastReword(repoRoot string, params []rewordParam) error {
	if len(params) < 1 {
		return nil
	}
	relinkBranches := func(repo *git.Repository, newCommitHash map[string]string, headDetached bool) error {
		it, err := repo.NewBranchIterator(git.BranchLocal)
		if err != nil {
			return err
		}
		if err = it.ForEach(func(b *git.Branch, _ git.BranchType) error {
			nh, ok := newCommitHash[b.Target().String()]
			if !ok {
				nh = b.Target().String()
			}
			nt, err := git.NewOid(nh)
			if err != nil {
				return err
			}
			_, err = b.SetTarget(nt, "")
			return err
		}); err != nil {
			return err
		}
		if headDetached {
			h, err := repo.Head()
			if err != nil {
				return err
			}
			nh, ok := newCommitHash[h.Target().String()]
			if !ok {
				nh = h.Target().String()
			}
			oid, err := git.NewOid(nh)
			if err != nil {
				return err
			}
			_, err = h.SetTarget(oid, "")
		}
		return err
	}

	log.Println("Updating commits...")
	repo, err := git.OpenRepository(repoRoot)
	if err != nil {
		return err
	}
	commits := make([]string, 0)
	for _, v := range params {
		commits = append(commits, v.hash)
	}
	g, err := buildCommitSubgraph(repoRoot, commits)
	if err != nil {
		return err
	}
	g.Reword(params)
	order := g.TopSort()
	newCommitHash := make(map[string]string)
	for _, v := range order {
		if !v.needsRebuild {
			continue
		}
		// TODO optimize loop body if needed
		oid, err := git.NewOid(v.id)
		if err != nil {
			return err
		}
		ocm, err := repo.LookupCommit(oid)
		if err != nil {
			return err
		}
		t, err := ocm.Tree()
		if err != nil {
			return err
		}
		parents := make([]*git.Commit, 0)
		for _, vv := range v.parents {
			id, err := git.NewOid(vv.id)
			if err != nil {
				return err
			}
			cm, err := repo.LookupCommit(id)
			if err != nil {
				return err
			}
			parents = append(parents, cm)
		}
		oid, err = repo.CreateCommit("", ocm.Author(), ocm.Committer(), v.message, t, parents...)
		if err != nil {
			return err
		}
		ocm, err = repo.LookupCommit(oid)
		if err != nil {
			return err
		}
		newCommitHash[v.id] = ocm.Id().String()
		v.id = ocm.Id().String()
	}
	log.Println("Relinking branches...")
	defer func() {
		exec.Command("/bin/sh", "-c", "cd", repoRoot, "&&", "git", "gc").Run()
	}()
	return relinkBranches(repo, newCommitHash, g.detachedHead)
}
