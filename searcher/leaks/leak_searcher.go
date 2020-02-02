package leaks

import (
	"fmt"
	"github.com/AlexAkulov/hungryfox/searcher/matching"
	"github.com/AlexAkulov/hungryfox/searcher/stats"
	"strings"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/entropy"
	"github.com/rs/zerolog"
)

type Matchers struct {
	Patterns []matching.PatternType
	Filters  []matching.PatternType
}

type LeakSearcher struct {
	LeakChannel  chan<- *hungryfox.Leak
	StatsChannel chan<- interface{}
	Matchers     *Matchers
	Log          zerolog.Logger
}

func (s *LeakSearcher) Process(diff *hungryfox.Diff) {
	leaks := s.getLeaks(diff, s.Matchers.Patterns)
	filteredLeaks := 0
	for i := range leaks {
		if filterLeak(leaks[i], s.Matchers.Filters) {
			filteredLeaks++
			continue
		}
		s.LeakChannel <- &leaks[i]
	}
	leaksCount := len(leaks) - filteredLeaks
	if leaksCount > 0 || filteredLeaks > 0 {
		s.StatsChannel <- stats.LeakStatsDiff{
			RepoURL:  diff.RepoURL,
			Found:    leaksCount,
			Filtered: filteredLeaks,
		}
	}
}

func (s *LeakSearcher) getLeaks(diff *hungryfox.Diff, patterns []matching.PatternType) []hungryfox.Leak {
	leaks := make([]hungryfox.Leak, 0)
	lines := strings.Split(diff.Content, "\n")
	for _, line := range lines {
		for _, pattern := range patterns {
			repoFilePath := fmt.Sprintf("%s/%s", diff.RepoURL, diff.FilePath)
			if !pattern.FileRe.MatchString(repoFilePath) {
				continue
			}
			if pattern.ContentRe.MatchString(line) {
				if pattern.Entropies != nil {
					if hasLowEntropy(line, pattern.Entropies) {
						s.Log.Debug().Str("leak", line).Msg("leak not matched because of low entropy")
						continue
					}
				}
				if len(line) > 1024 {
					line = line[:1024]
				}
				leaks = append(leaks, hungryfox.Leak{
					RepoPath:     diff.RepoPath,
					FilePath:     diff.FilePath,
					PatternName:  pattern.Name,
					Regexp:       pattern.ContentRe.String(),
					LeakString:   line,
					CommitHash:   diff.CommitHash,
					TimeStamp:    diff.TimeStamp,
					CommitAuthor: diff.Author,
					CommitEmail:  diff.AuthorEmail,
					RepoURL:      diff.RepoURL,
				})
			}
		}
	}
	return leaks
}

func hasLowEntropy(line string, entropies *matching.EntropyType) bool {
	if entropies.WordMin > 0 {
		wordEntropy := entropy.GetWordShannonEntropy(line)
		if wordEntropy >= entropies.WordMin {
			return false
		}
	}
	if entropies.LineMin > 0 {
		entropy := entropy.GetShannonEntropy(line)
		if entropy >= entropies.LineMin {
			return false
		}
	}
	return entropies.WordMin > 0 || entropies.LineMin > 0
}

func filterLeak(leak hungryfox.Leak, filters []matching.PatternType) bool {
	for _, filter := range filters {
		if filter.FileRe.MatchString(fmt.Sprintf("%s/%s", leak.RepoURL, leak.FilePath)) && filter.ContentRe.MatchString(leak.LeakString) {
			return true
		}
	}
	return false
}
