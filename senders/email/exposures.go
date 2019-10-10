package email

import (
	"fmt"

	"github.com/AlexAkulov/hungryfox"
	"github.com/facebookgo/muster"
)

type exposuresBatch struct {
	ExposuresCount int
	Repos          map[string]*repoExposures
	Files          map[string]struct{}
	Sender         *Sender
}

type exposuresMailTemplateData struct {
	ExposuresCount int
	FilesCount     int
	Repos          []*repoExposures
}

type repoExposures struct {
	RepoURL string
	Items   []hungryfox.VulnerableDependency
}

func (s *Sender) exposuresBatchMaker() muster.Batch {
	return &exposuresBatch{
		Repos:  make(map[string]*repoExposures),
		Files:  make(map[string]struct{}),
		Sender: s,
	}
}

func (b *exposuresBatch) Fire(notifier muster.Notifier) {
	defer notifier.Done()
	if b.ExposuresCount < 1 {
		return
	}
	messageData := &exposuresMailTemplateData{
		FilesCount:     len(b.Files),
		ExposuresCount: b.ExposuresCount,
	}
	for _, repo := range b.Repos {
		messageData.Repos = append(messageData.Repos, repo)
	}
	err := b.Sender.sendMessage(b.Sender.AuditorEmail, getExposuresSubject(messageData), messageData)
	if err != nil {
		b.Sender.Log.Error().Str("error", err.Error()).Msg("can't send email")
	}
}

func (b *exposuresBatch) Add(item interface{}) {
	dep, ok := item.(hungryfox.VulnerableDependency)
	if !ok {
		return
	}
	if b.Repos[dep.RepoURL] == nil {
		b.Repos[dep.RepoURL] = &repoExposures{
			RepoURL: dep.RepoURL,
			Items:   []hungryfox.VulnerableDependency{},
		}
	}
	b.Repos[dep.RepoURL].Items = append(b.Repos[dep.RepoURL].Items, dep)
	b.Files[fmt.Sprintf("%s/%s", dep.RepoURL, dep.FilePath)] = struct{}{}
	b.ExposuresCount++
}

func getExposuresSubject(messageData *exposuresMailTemplateData) string {
	if len(messageData.Repos) == 1 {
		return fmt.Sprintf("Found %d vulnerabilities in %s", messageData.ExposuresCount, messageData.Repos[0].RepoURL)
	} else {
		return fmt.Sprintf("Found %d vulnerabilities in %d repos", messageData.ExposuresCount, len(messageData.Repos))
	}
}
