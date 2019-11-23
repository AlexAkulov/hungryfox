package searcher

import (
	"fmt"
	"strings"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/entropy"
	"github.com/rs/zerolog"
)

type Matchers struct {
	patterns []patternType
	filters  []patternType
}

type LeakAnalyzer struct {
	LeakChannel  chan<- *hungryfox.Leak
	StatsChannel chan<- statsDiff
	Matchers     *Matchers
	Log          zerolog.Logger
}

func (a *LeakAnalyzer) Analyze(diff *hungryfox.Diff) {
	leaks := a.getLeaks(diff, a.Matchers.patterns)
	filteredLeaks := 0
	for i := range leaks {
		if filterLeak(leaks[i], a.Matchers.filters) {
			filteredLeaks++
			continue
		}
		a.LeakChannel <- &leaks[i]
	}
	leaksCount := len(leaks) - filteredLeaks
	if leaksCount > 0 || filteredLeaks > 0 {
		a.StatsChannel <- statsDiff{
			Kind:     LeakStat,
			RepoURL:  diff.RepoURL,
			Found:    leaksCount,
			Filtered: filteredLeaks,
		}
	}
}

func (a *LeakAnalyzer) getLeaks(diff *hungryfox.Diff, patterns []patternType) []hungryfox.Leak {
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
						a.Log.Debug().Str("leak", line[:100]).Msg("leak not matched because of low entropy")
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

func hasLowEntropy(line string, entropies *entropyType) bool {
	isLowWord, isLowLine := false, false
	if entropies.WordMin > 0 {
		wordEntropy := entropy.GetWordShannonEntropy(line)
		isLowWord = wordEntropy < entropies.WordMin
	}
	if entropies.LineMin > 0 {
		entropy := entropy.GetShannonEntropy(line)
		isLowLine = entropy < entropies.LineMin
	}
	return isLowWord && isLowLine
}

func filterLeak(leak hungryfox.Leak, filters []patternType) bool {
	for _, filter := range filters {
		if filter.FileRe.MatchString(fmt.Sprintf("%s/%s", leak.RepoURL, leak.FilePath)) && filter.ContentRe.MatchString(leak.LeakString) {
			return true
		}
	}
	return false
}
