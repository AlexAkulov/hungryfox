package gitlab

import (
	"net/http"

	"github.com/AlexAkulov/hungryfox"
	"github.com/xanzy/go-gitlab"
)

type Client struct {
	URL     string
	Token   string
	WorkDir string
	client  *gitlab.Client
}

type FetchOptions struct {
	ExcludeNamespaces []string
	ExcludeProjects   []string
	Search            string
}

func (c *Client) FetchGroupRepos(options *FetchOptions) ([]hungryfox.RepoLocation, error) {
	c.connect()
	isSimple := true
	search := options.Search
	listOptions := &gitlab.ListProjectsOptions{
		Simple: &isSimple,
		Search: &search,
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}
	excludedNamespaces := toMap(options.ExcludeNamespaces)
	excludedProjects := toMap(options.ExcludeProjects)
	var locations []hungryfox.RepoLocation

	for {
		projects, response, err := c.client.Projects.ListProjects(listOptions)
		if err != nil {
			return locations, err
		}

		for _, proj := range projects {
			if !excludedNamespaces[proj.Namespace.Name] && !excludedProjects[proj.PathWithNamespace] {
				locations = append(locations, *c.toRepoLocation(proj))
			}
		}
		if response.NextPage == 0 {
			break
		}
		listOptions.ListOptions.Page = response.NextPage
	}

	return locations, nil
}

func (c *Client) connect() {
	if c.client == nil {
		c.client = gitlab.NewClient(&http.Client{}, c.Token)
		c.client.SetBaseURL(c.URL)
	}
}

func (c *Client) toRepoLocation(proj *gitlab.Project) *hungryfox.RepoLocation {
	return &hungryfox.RepoLocation{
		CloneURL: proj.SSHURLToRepo,
		URL:      proj.WebURL,
		DataPath: c.WorkDir,
		RepoPath: proj.PathWithNamespace,
	}
}

func toMap(arr []string) map[string]bool {
	stringMap := make(map[string]bool)
	for _, str := range arr {
		stringMap[str] = true
	}
	return stringMap
}
