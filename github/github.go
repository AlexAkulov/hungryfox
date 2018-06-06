package github

import (
	"context"
	"net/http"

	"github.com/AlexAkulov/hungryfox"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Client struct {
	Token   string
	WorkDir string
	client  *github.Client
}

func (c *Client) connect() {
	if c.client == nil {
		c.client = github.NewClient(c.getTokenClient())
	}
}

func (c *Client) FetchOrgRepos(orgName string) ([]hungryfox.RepoLocation, error) {
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}
	c.connect()
	ctx := context.Background()
	var repoList []hungryfox.RepoLocation

	for {
		repo, resp, err := c.client.Repositories.ListByOrg(ctx, orgName, opts)
		if err != nil {
			return repoList, err
		}
		if resp.NextPage == 0 {
			break
		}
		repoList = append(repoList, c.convertRepoList(repo)...)
		opts.Page = resp.NextPage
	}
	return repoList, nil
}

func (c *Client) FetchUserRepos(userName string) ([]hungryfox.RepoLocation, error) {
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}
	c.connect()
	ctx := context.Background()
	var repoList []hungryfox.RepoLocation
	for {
		repo, resp, err := c.client.Repositories.List(ctx, userName, opts)
		if err != nil {
			return repoList, err
		}
		if resp.NextPage == 0 {
			break
		}
		repoList = append(repoList, c.convertRepoList(repo)...)
		opts.Page = resp.NextPage
	}
	return repoList, nil
}

func (c *Client) convertRepoList(list []*github.Repository) (hfRepoList []hungryfox.RepoLocation) {
	for _, repo := range list {
		hfRepoList = append(hfRepoList, hungryfox.RepoLocation{
			URL:      *repo.HTMLURL,
			CloneURL: *repo.CloneURL,
			DataPath: c.WorkDir,
			RepoPath: *repo.FullName,
		})
	}
	return
}

func (c *Client) getTokenClient() *http.Client {
	if c.Token == "" {
		return nil
	}
	return oauth2.NewClient(
		context.Background(),
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token}),
	)
}
