package scanmanager

import (
	"strconv"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/gitlab"
)

func (sm *ScanManager) inspectGitlab(inspect config.Inspect) error {
	sm.Log.Debug().Msg("start fetching gitlab projects")
	gitlabClient := gitlab.Client{
		Token:   inspect.Token,
		URL:     inspect.GitlabURL,
		WorkDir: inspect.WorkDir,
	}
	repoLocations := make(map[hungryfox.RepoLocation]struct{})

	fetchOpts := &gitlab.FetchOptions{
		ExcludeNamespaces: inspect.GitlabExcludeNamespaces,
		ExcludeProjects:   inspect.GitlabExcludeProjects,
		Search:            inspect.GitlabFilter,
	}
	locations, err := gitlabClient.FetchGroupRepos(fetchOpts)
	if err != nil {
		sm.Log.Error().Msg(err.Error())
		return err
	}
	sm.Log.Debug().Str("count", strconv.Itoa(len(locations))).Msg("finished fetching gitlab projects")
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
