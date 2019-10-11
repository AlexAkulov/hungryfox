package email

import (
	"fmt"

	"github.com/AlexAkulov/hungryfox"
	"github.com/facebookgo/muster"
)

type exposuresBatch struct {
	Exposures []hungryfox.VulnerableDependency
	Sender    *Sender
}

type exposuresMailTemplateData struct {
	ExposuresCount int
	Repos          []*repoExposures
}

type repoExposures struct {
	RepoURL string
	Items   []hungryfox.VulnerableDependency
}

func (s *Sender) exposuresBatchMaker() muster.Batch {
	return &exposuresBatch{
		Sender: s,
	}
}

func (b *exposuresBatch) Add(item interface{}) {
	dep, ok := item.(hungryfox.VulnerableDependency)
	if !ok {
		return
	}
	b.Exposures = append(b.Exposures, dep)
}

func addOrAppend(repos map[string]*repoExposures, exp hungryfox.VulnerableDependency) {
	if repo, ok := repos[exp.RepoURL]; ok {
		repo.Items = append(repo.Items, exp)
	} else {
		repos[exp.RepoURL] = &repoExposures{
			RepoURL: exp.RepoURL,
			Items:   []hungryfox.VulnerableDependency{exp},
		}
	}
}

func (b *exposuresBatch) Fire(notifier muster.Notifier) {
	defer notifier.Done()
	if len(b.Exposures) < 1 {
		return
	}
	allRepos := make(map[string]*repoExposures)
	reposByAuthor := make(map[string]map[string]*repoExposures)
	for _, exp := range b.Exposures {
		addOrAppend(allRepos, exp)

		if b.Sender.Config.SendToAuthor {
			if repo, ok := reposByAuthor[exp.CommitEmail]; ok {
				addOrAppend(repo, exp)
			} else {
				reposByAuthor[exp.CommitEmail] = make(map[string]*repoExposures)
				addOrAppend(reposByAuthor[exp.CommitEmail], exp)
			}
		}
	}

	b.Sender.sendExposuresMail(b.Sender.AuditorEmail, allRepos)
	if len(reposByAuthor) > 0 {
		for authorsEmail, repos := range reposByAuthor {
			b.Sender.sendExposuresMail(authorsEmail, repos)
		}
	}
}

func (s *Sender) sendExposuresMail(recipients string, repos map[string]*repoExposures) {
	messageData := makeTemplateData(repos)
	err := s.sendMessage(recipients, getExposuresSubject(messageData), messageData)
	if err != nil {
		s.Log.Error().Str("error", err.Error()).Msg("can't send email")
	}
}

func makeTemplateData(reposMap map[string]*repoExposures) *exposuresMailTemplateData {
	var repos []*repoExposures
	expsCount := 0
	for _, repo := range reposMap {
		repos = append(repos, repo)
		for _, dep := range repo.Items {
			expsCount += len(dep.Vulnerabilities)
		}
	}
	return &exposuresMailTemplateData{
		ExposuresCount: expsCount,
		Repos:          repos,
	}
}

func getExposuresSubject(messageData *exposuresMailTemplateData) string {
	vulnsWord := "vulnerabilities"
	if messageData.ExposuresCount == 1 {
		vulnsWord = "vulnerability"
	}
	if len(messageData.Repos) == 1 {
		return fmt.Sprintf("Found %d %s in %s", messageData.ExposuresCount, vulnsWord, messageData.Repos[0].RepoURL)
	} else {
		return fmt.Sprintf("Found %d %s in %d repos", messageData.ExposuresCount, vulnsWord, len(messageData.Repos))
	}
}
