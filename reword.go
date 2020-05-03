package main

import (
	git "github.com/libgit2/git2go/v28"
)

func fastReword(repoRoot string, params []rewordParam) error {
	relinkBranches := func(repo *git.Repository, newCommitHash map[string]string) error {
		it, err := repo.NewBranchIterator(git.BranchLocal)
		if err != nil {
			return err
		}
		return it.ForEach(func(b *git.Branch, _ git.BranchType) error {
			nt, err := git.NewOid(newCommitHash[b.Target().String()])
			if err != nil {
				return err
			}
			_, err = b.SetTarget(nt, "")
			return err
		})
	}
	// TODO detached head is not detected and updated, needs to be fixed, also in graph
	repo, err := git.OpenRepository(repoRoot)
	if err != nil {
		return err
	}
	g, err := buildCommitGraph(repoRoot)
	if err != nil {
		return err
	}
	g.Reword(params)
	order := g.TopSort()
	newCommitHash := make(map[string]string)
	// TODO we work with full graph now, but for subgraph we need to detect if branch really has no parents
	for _, v := range order {
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
	if err = relinkBranches(repo, newCommitHash); err != nil {
		return err
	}
	return nil
}
