package scanmanager

import (
	"fmt"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/github"
)

func getGitHubRepoURL(repoPath string) string {
	return fmt.Sprintf("https://github.com/%s", repoPath)
}
func getGitHubCloneURL(repoPath string) string {
	return fmt.Sprintf("https://github.com/%s.git", repoPath)
}
func getGitHubRepoPath(repoPath string) (string, error) {
	return "", nil
}

func (sm *ScanManager) inspectGithub(inspect config.Inspect) error {
	githubClient := github.Client{
		Token:   inspect.Token,
		WorkDir: inspect.WorkDir,
	}
	repoLocations := map[hungryfox.RepoLocation]struct{}{}

	for _, org := range inspect.Orgs {
		sm.Log.Debug().Str("organisation", org).Msg("get repos from github.com")
		repoList, err := githubClient.FetchOrgRepos(org)
		if err != nil {
			sm.Log.Error().Str("error", err.Error()).Str("organisation", org).Msg("can't fetch repos from github")
			continue
		}
		for _, repoLocation := range repoList {
			repoLocations[repoLocation] = struct{}{}
		}
	}
	for _, user := range inspect.Users {
		sm.Log.Debug().Str("user", user).Msg("get repos from github.com")
		repoList, err := githubClient.FetchUserRepos(user)
		if err != nil {
			sm.Log.Error().Str("error", err.Error()).Str("user", user).Msg("can't fetch repos from github")
			continue
		}
		for _, repoLocation := range repoList {
			repoLocations[repoLocation] = struct{}{}
		}
	}
	for _, repo := range inspect.Repos {
		repoLocation := hungryfox.RepoLocation{
			URL:      getGitHubRepoURL(repo),
			CloneURL: getGitHubCloneURL(repo),
			RepoPath: repo,
			DataPath: inspect.WorkDir,
		}
		repoLocations[repoLocation] = struct{}{}
	}

	for repoLocation := range repoLocations {
		sm.repoList.AddRepo(hungryfox.Repo{
			Location: repoLocation,
			Options:  hungryfox.RepoOptions{AllowUpdate: true},
		})
	}

	return nil
}
