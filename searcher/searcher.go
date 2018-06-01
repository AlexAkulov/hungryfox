package searcher

import (
	"fmt"
	"strings"
	"sync"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"

	"gopkg.in/tomb.v2"
)

type RepoStats struct {
	LeaksFound   int `json:"leaks_found"`
	LeaksFiltred int `json:"leaks_filtred"`
}

type Searcher struct {
	Workers     int
	DiffChannel <-chan *hungryfox.Diff
	LeakChannel chan<- *hungryfox.Leak
	config      *config.Config
	stats       map[string]RepoStats
	statsMutex  sync.RWMutex
	tomb        tomb.Tomb
}

func (s *Searcher) Start() error {
	if s.Workers < 1 {
		return fmt.Errorf("workers count can't be less 1")
	}

	s.stats = map[string]RepoStats{}
	for i := 0; i < s.Workers; i++ {
		s.tomb.Go(s.worker)
	}
	return nil
}

func (s *Searcher) worker() error {
	for {
		select {
		case <-s.tomb.Dying():
			return nil
		case diff := <-s.DiffChannel:
			leaks := s.GetLeaks(diff)
			filtredLeaks := 0
			for _, leak := range leaks {
				if ok, _ := s.filterLeak(leak); ok {
					filtredLeaks++
					continue
				}
				s.LeakChannel <- leak
			}
			leaksCount := len(leaks) - filtredLeaks
			if leaksCount > 0 || filtredLeaks > 0 {
				s.statsMutex.Lock()
				repoStats, _ := s.stats[diff.RepoURL]
				repoStats.LeaksFiltred += filtredLeaks
				repoStats.LeaksFound += leaksCount
				s.stats[diff.RepoURL] = repoStats
				s.statsMutex.Unlock()
			}
		}
	}
}

func (s *Searcher) Stop() error {
	s.tomb.Kill(nil)
	return s.tomb.Wait()
}

func (s *Searcher) SetConfig(config *config.Config) {
	s.config = config
}

func (s *Searcher) Status(repoURL string) RepoStats {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()
	if repoStats, ok := s.stats[repoURL]; ok {
		return repoStats
	}
	return RepoStats{}
}

func (s *Searcher) GetLeaks(diff *hungryfox.Diff) []*hungryfox.Leak {
	leaks := make([]*hungryfox.Leak, 0)
	lines := strings.Split(diff.Content, "\n")
	for _, line := range lines {
		for _, pattern := range s.config.Patterns {
			repoFilePath := fmt.Sprintf("%s/%s", diff.RepoURL, diff.FilePath)
			if !pattern.FileRe.MatchString(repoFilePath) {
				continue
			}
			if pattern.ContentRe.MatchString(line) {
				if len(line) > 1024 {
					line = line[:1024]
				}
				leak := &hungryfox.Leak{
					RepoPath:     diff.RepoPath,
					FilePath:     diff.FilePath,
					PatternName:  pattern.Name,
					Regexp:       pattern.Content,
					LeakString:   line,
					CommitHash:   diff.CommitHash,
					TimeStamp:    diff.TimeStamp,
					CommitAuthor: diff.Author,
					CommitEmail:  diff.AuthorEmail,
					RepoURL:      diff.RepoURL,
				}
				leaks = append(leaks, leak)
			}
		}
	}
	return leaks
}

func (s *Searcher) filterLeak(leak *hungryfox.Leak) (bool, *config.Pattern) {
	for _, filter := range s.config.Filters {
		if filter.FileRe.MatchString(fmt.Sprintf("%s/%s", leak.RepoURL, leak.FilePath)) && filter.ContentRe.MatchString(leak.LeakString) {
			return true, filter
		}
	}
	return false, nil
}
