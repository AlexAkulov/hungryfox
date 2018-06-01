package repo

import (
	"os"
	"path/filepath"

	"gopkg.in/src-d/go-git.v4"
)

func (r *Repo) fullRepoPath() string {
	return filepath.Join(r.DataPath, r.RepoPath)
}

func (r *Repo) CloneOrUpdate() error {
	if _, err := os.Stat(r.fullRepoPath()); os.IsNotExist(err) {
		// s.Log.Debug().Str("repo", repo).Str("path", repoFullPath).Msg("path is not exists, try to create")
		if err := os.MkdirAll(r.fullRepoPath(), 755); err != nil {
			return err
		}
		cloneOptions := &git.CloneOptions{
			URL:   r.URL,
		}
		// s.Log.Debug().Str("repo", repo).Str("path", repoFullPath).Msg("try to clone")
		repository, err := git.PlainClone(r.fullRepoPath(), false, cloneOptions)
		if err != nil {
			return err
		}
		r.repository = repository
		return nil 
	}

	if err := r.Open(); err != nil {
		return err
	}

	if err := r.repository.Fetch(&git.FetchOptions{Force: true}); err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	return nil
}
