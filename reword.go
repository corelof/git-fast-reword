package main

import (
	git "github.com/libgit2/git2go/v28"
)

// TODO add logs
// TODO call git garbage collector after job
// TODO improve test corner cases coverage

func fastReword(repoRoot string, params []rewordParam) error {
	relinkBranches := func(repo *git.Repository, newCommitHash map[string]string, headDetached bool) error {
		it, err := repo.NewBranchIterator(git.BranchLocal)
		if err != nil {
			return err
		}
		if err = it.ForEach(func(b *git.Branch, _ git.BranchType) error {
			nt, err := git.NewOid(newCommitHash[b.Target().String()])
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
			oid, err := git.NewOid(newCommitHash[h.Target().String()])
			if err != nil {
				return err
			}
			_, err = h.SetTarget(oid, "")
		}
		return err
	}

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
		// TODO optimize loop body
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
	return relinkBranches(repo, newCommitHash, g.detachedHead)
}
