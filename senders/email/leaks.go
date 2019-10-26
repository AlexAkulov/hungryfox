package email

import (
	"fmt"
	"strings"

	"github.com/AlexAkulov/hungryfox"

	"github.com/facebookgo/muster"
)

type mailTemplateStruct struct {
	LeaksCount int
	FilesCount int
	Repos      []*mailTemplateRepoStruct
}

type mailTemplateRepoStruct struct {
	RepoURL string
	Items   []hungryfox.Leak
}

func (s *Sender) leakBatchMaker() muster.Batch {
	return &leakBatch{
		Sender: s,
		Repos:  map[string]*mailTemplateRepoStruct{},
		Files:  map[string]struct{}{},
	}
}

type leakBatch struct {
	LeaksCount int
	Repos      map[string]*mailTemplateRepoStruct
	Files      map[string]struct{}
	Sender     *Sender
}

func (b *leakBatch) Add(item interface{}) {
	leak, ok := item.(hungryfox.Leak)
	if !ok {
		return
	}
	normalizeLeakString(&leak)
	if b.Repos[leak.RepoURL] == nil {
		b.Repos[leak.RepoURL] = &mailTemplateRepoStruct{
			RepoURL: leak.RepoURL,
			Items:   []hungryfox.Leak{},
		}
	}
	b.Repos[leak.RepoURL].Items = append(b.Repos[leak.RepoURL].Items, leak)
	b.Files[fmt.Sprintf("%s/%s", leak.RepoURL, leak.FilePath)] = struct{}{}
	b.LeaksCount++
}

func (b *leakBatch) Fire(notifier muster.Notifier) {
	defer notifier.Done()
	if b.LeaksCount < 1 {
		return
	}
	messageData := &mailTemplateStruct{
		FilesCount: len(b.Files),
		LeaksCount: b.LeaksCount,
	}
	for _, repo := range b.Repos {
		messageData.Repos = append(messageData.Repos, repo)
	}
	err := b.Sender.sendMessage(b.Sender.AuditorEmail, getLeaksSubject(messageData), *messageData)
	if err != nil {
		b.Sender.Log.Error().Str("error", err.Error()).Msg("can't send email")
	}
}

func normalizeLeakString(leak *hungryfox.Leak) {
	leak.LeakString = strings.TrimSpace(leak.LeakString)
	if len(leak.LeakString) > 512 {
		leak.LeakString = "too long"
	}
}

func getLeaksSubject(messageData *mailTemplateStruct) string {
	if len(messageData.Repos) == 1 {
		return fmt.Sprintf("Found %d leaks in %s", messageData.LeaksCount, messageData.Repos[0].RepoURL)
	} else {
		return fmt.Sprintf("Found %d leaks in %d repos", messageData.LeaksCount, len(messageData.Repos))
	}
}
