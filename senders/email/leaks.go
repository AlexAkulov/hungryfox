package email

import (
	"fmt"
	"strings"

	"github.com/AlexAkulov/hungryfox"

	"github.com/facebookgo/muster"
)

type leakBatch struct {
	Leaks  []hungryfox.Leak
	Sender *Sender
}

type repoLeaks struct {
	RepoURL string
	Items   []hungryfox.Leak
	Files   map[string]struct{}
}

type leaksMailTemplateData struct {
	LeaksCount int
	FilesCount int
	Repos      []*repoLeaks
}

func (s *Sender) leakBatchMaker() muster.Batch {
	return &leakBatch{
		Sender: s,
	}
}

func (b *leakBatch) Add(item interface{}) {
	leak, ok := item.(hungryfox.Leak)
	if !ok {
		return
	}
	normalizeLeakString(&leak)
	b.Leaks = append(b.Leaks, leak)
}

func addOrAppendLeak(repos map[string]*repoLeaks, leak hungryfox.Leak) {
	filename := fmt.Sprintf("%s/%s", leak.RepoURL, leak.FilePath)
	if repo, ok := repos[leak.RepoURL]; ok {
		repo.Items = append(repo.Items, leak)
		repo.Files[filename] = struct{}{}
	} else {
		files := make(map[string]struct{})
		files[filename] = struct{}{}
		repos[leak.RepoURL] = &repoLeaks{
			RepoURL: leak.RepoURL,
			Items:   []hungryfox.Leak{leak},
			Files:   files,
		}
	}
}

func (b *leakBatch) Fire(notifier muster.Notifier) {
	defer notifier.Done()
	if len(b.Leaks) < 1 {
		return
	}
	allRepos := make(map[string]*repoLeaks)
	reposByAuthor := make(map[string]map[string]*repoLeaks)

	for _, leak := range b.Leaks {
		addOrAppendLeak(allRepos, leak)

		if b.Sender.Config.SendToAuthor {
			if repo, ok := reposByAuthor[leak.CommitEmail]; ok {
				addOrAppendLeak(repo, leak)
			} else {
				reposByAuthor[leak.CommitEmail] = make(map[string]*repoLeaks)
				addOrAppendLeak(reposByAuthor[leak.CommitEmail], leak)
			}
		}
	}

	b.Sender.sendLeaksMail(b.Sender.AuditorEmail, allRepos)
	if len(reposByAuthor) > 0 {
		for authorsEmail, repos := range reposByAuthor {
			if b.Sender.isOkRecipient(authorsEmail) {
				b.Sender.sendLeaksMail(authorsEmail, repos)
			} else {
				b.Sender.Log.Warn().Str("email", authorsEmail).Msg("recipient doesn't match specified pattern and won't receive a notification")
			}
		}
	}
}

func (s *Sender) sendLeaksMail(recipients string, repos map[string]*repoLeaks) {
	messageData := makeLeaksTemplateData(repos)
	err := s.sendMessage(recipients, getLeaksSubject(messageData), *messageData)
	if err != nil {
		s.Log.Error().Str("error", err.Error()).Msg("can't send email")
	}
}

func makeLeaksTemplateData(reposMap map[string]*repoLeaks) *leaksMailTemplateData {
	var repos []*repoLeaks
	leaksCount, filesCount := 0, 0
	for _, repo := range reposMap {
		repos = append(repos, repo)
		leaksCount += len(repo.Items)
		filesCount += len(repo.Files)
	}
	return &leaksMailTemplateData{
		LeaksCount: leaksCount,
		FilesCount: filesCount,
		Repos:      repos,
	}
}

func normalizeLeakString(leak *hungryfox.Leak) {
	leak.LeakString = strings.TrimSpace(leak.LeakString)
	if len(leak.LeakString) > 512 {
		leak.LeakString = "too long"
	}
}

func getLeaksSubject(messageData *leaksMailTemplateData) string {
	if len(messageData.Repos) == 1 {
		return fmt.Sprintf("Found %d leaks in %s", messageData.LeaksCount, messageData.Repos[0].RepoURL)
	} else {
		return fmt.Sprintf("Found %d leaks in %d repos", messageData.LeaksCount, len(messageData.Repos))
	}
}
