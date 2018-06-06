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

func getGitHubRepoPath(repoPath string) (string, error) {
	return "", nil
}

func (sm *ScanManager) inspectGithub(inspect *config.Inspect) error {
	githubClient := github.Client{
		Token:   inspect.Token,
		WorkDir: inspect.WorkDir,
	}
	for _, org := range inspect.Orgs {
		repoList, err := githubClient.FetchOrgRepos(org)
		if err != nil {
			sm.Log.Error().Str("error", err.Error()).Str("organisation", org).Msg("can't fetch repos from github")
			continue
		}
		for i := range repoList {
			sm.repoList.AddRepo(repoList[i])
		}
	}
	for _, user := range inspect.Users {
		repoList, err := githubClient.FetchUserRepos(user)
		if err != nil {
			sm.Log.Error().Str("error", err.Error()).Str("user", user).Msg("can't fetch repos from github")
			continue
		}
		for i := range repoList {
			sm.repoList.AddRepo(repoList[i])
		}
	}
	for _, repo := range inspect.Repos {
		sm.repoList.AddRepo(hungryfox.Repo{
			Location: hungryfox.RepoLocation{
				URL:      getGitHubRepoURL(repo),
				RepoPath: repo,
				DataPath: inspect.WorkDir,
			},
			Options: hungryfox.RepoOptions{
				AllowUpdate: true,
			},
		})
	}
	return nil
}
