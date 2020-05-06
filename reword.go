package main

import (
	"log"
	"os/exec"

	git "github.com/libgit2/git2go/v28"
)

// TODO stress on intellij repo

func relinkBranches(repo *git.Repository, newCommitHash map[string]string, headDetached bool) error {
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

func relinkTags(repo *git.Repository, newCommitHash map[string]string) error {
	tags := make([]*git.Tag, 0)
	lightweight := make([]string, 0)

	if err := repo.Tags.Foreach(func(name string, obj *git.Oid) error {
		tag, err := repo.LookupTag(obj)
		if err != nil {
			lightweight = append(lightweight, name)
			return nil
		}
		tags = append(tags, tag)
		return nil
	}); err != nil {
		return err
	}

	for _, tag := range tags {
		newTarget, ok := newCommitHash[tag.Target().Id().String()]
		if !ok {
			newTarget = tag.Target().Id().String()
		}
		if newTarget == tag.Target().Id().String() {
			continue
		}
		name := tag.Name()
		tagger := tag.Tagger()
		message := tag.Message()
		oid, err := git.NewOid(newTarget)
		if err != nil {
			return err
		}
		cm, err := repo.LookupCommit(oid)
		if err != nil {
			return err
		}
		if err := repo.Tags.Remove(name); err != nil {
			return err
		}
		if _, err = repo.Tags.Create(name, cm, tagger, message); err != nil {
			return err
		}
	}

	for _, v := range lightweight {
		ref, err := repo.References.Lookup(v)
		if err != nil {
			return err
		}
		newTarget, ok := newCommitHash[ref.Target().String()]
		if !ok {
			newTarget = ref.Target().String()
		}
		if newTarget == ref.Target().String() {
			continue
		}

		oid, err := git.NewOid(newTarget)
		if err != nil {
			return err
		}
		if _, err = ref.SetTarget(oid, ""); err != nil {
			return err
		}
	}
	return nil
}

func fastReword(repo *git.Repository, params []rewordParam, dateOptimization, headOnly bool) error {
	if len(params) < 1 {
		return nil
	}

	commits := make([]string, 0)
	for _, v := range params {
		commits = append(commits, v.hash)
	}
	log.Println("Building repo graph...")
	g, err := buildCommitSubgraph(repo, commits, dateOptimization, headOnly)
	if err != nil {
		return err
	}
	g.Reword(params)
	order := g.TopSort()
	log.Printf("Commits to rebuild: %d", len(order))
	newCommitHash := make(map[string]string)
	log.Println("Updating commits...")
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
	if err = relinkBranches(repo, newCommitHash, g.detachedHead); err != nil {
		return err
	}
	log.Println("Relinking tags...")
	if err = relinkTags(repo, newCommitHash); err != nil {
		return err
	}
	log.Println("Running git garbage collector...")
	return exec.Command("/bin/sh", "-c", "cd", repo.Path(), "&&", "git", "gc").Run()
}
