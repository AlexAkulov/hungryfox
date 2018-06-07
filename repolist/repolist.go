package repolist

import (
	"time"

	"github.com/AlexAkulov/hungryfox"
)

type RepoList struct {
	list []hungryfox.Repo
	State hungryfox.IStateManager
}

func (l *RepoList) Clear() {
	l.list = nil
}

func (l *RepoList) addRepo(r hungryfox.Repo) {
	if l.list == nil {
		l.list = make([]hungryfox.Repo, 0)
	}
	for i := range l.list {
		if l.list[i].Location.URL == r.Location.URL {
			l.list[i] = r
			return
		}
	}
	l.list = append(l.list, r)
}

func (l *RepoList) AddRepo(r hungryfox.Repo) {
	r.State, r.Scan = l.State.Load(r.Location.URL)
	l.addRepo(r)
}

func (l *RepoList) UpdateRepo(r hungryfox.Repo) {
	l.addRepo(r)
	l.State.Save(r)
}

func (l *RepoList) GetRepoByIndex(i int) *hungryfox.Repo {
	if i > len(l.list)-1 || i < 0 {
		return nil
	}
	r := l.list[i]
	return &r
}

func (l *RepoList) GetRepoForScan() int {
	rID := -1
	lastScan := time.Now().UTC()
	for i, r := range l.list {
		if r.Scan.StartTime.IsZero() {
			return i
		}
		if r.Scan.EndTime.Before(lastScan) {
			rID = i
			lastScan = r.Scan.EndTime
		}
	}
	return rID
}

func (l *RepoList) GetTotalRepos() int {
	return len(l.list)
}
