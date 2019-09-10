package scanmanager

import (
	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/gitlab"
)

func (sm *ScanManager) inspectGitlab(inspect config.Inspect) error {
	gitlabClient := gitlab.Client{
		Token:   inspect.Token,
		URL:     inspect.GitlabURL,
		WorkDir: inspect.WorkDir,
	}
	var repoLocations map[hungryfox.RepoLocation]struct{}

	fetchOpts := &gitlab.FetchOptions{
		ExcludeNamespaces: &inspect.ExcludeNamespaces,
	}
	locations, err := gitlabClient.FetchGroupRepos(fetchOpts)
	if err != nil {
		return err
	}
	for _, location := range locations {
		repoLocations[location] = struct{}{}
	}
	for repoLocation := range repoLocations {
		sm.repoList.AddRepo(hungryfox.Repo{
			Location: repoLocation,
			Options:  hungryfox.RepoOptions{AllowUpdate: true},
		})
	}
	return nil
}
