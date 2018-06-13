package searcher

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	sync "github.com/sasha-s/go-deadlock"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"

	"github.com/rs/zerolog"
	"gopkg.in/tomb.v2"
	yaml "gopkg.in/yaml.v2"
)

var matchAllRegex = regexp.MustCompile(".+")

type patternType struct {
	Name      string
	ContentRe *regexp.Regexp
	FileRe    *regexp.Regexp
}

type RepoStats struct {
	LeaksFound   int `json:"leaks_found"`
	LeaksFiltred int `json:"leaks_filtred"`
}

type Searcher struct {
	Workers     int
	DiffChannel <-chan *hungryfox.Diff
	LeakChannel chan<- *hungryfox.Leak
	Log         zerolog.Logger

	config           *config.Config
	stats            map[string]RepoStats
	statsMutex       sync.RWMutex
	tomb             tomb.Tomb
	patterns         []patternType
	filters          []patternType
	updateConfigChan chan *config.Config
}

func compilePatterns(configPatterns []config.Pattern) ([]patternType, error) {
	result := make([]patternType, 0)
	for _, configPattern := range configPatterns {
		p := patternType{
			Name:      configPattern.Name,
			FileRe:    matchAllRegex,
			ContentRe: matchAllRegex,
		}
		if configPattern.File != "*" && configPattern.File != "" {
			var err error
			if p.FileRe, err = regexp.Compile(configPattern.File); err != nil {
				return nil, fmt.Errorf("can't compile pattern file regexp '%s' with: %v", configPattern.File, err)
			}
		}
		if configPattern.Content != "*" && configPattern.Content != "" {
			var err error
			if p.ContentRe, err = regexp.Compile(configPattern.Content); err != nil {
				return nil, fmt.Errorf("can't compile pattern content regexp '%s' with: %v", configPattern.Content, err)
			}
		}
		result = append(result, p)
	}
	return result, nil
}

func (s *Searcher) Update(conf *config.Config) {
	s.updateConfigChan <- conf
}

func (s *Searcher) Start(conf *config.Config) error {
	if err := s.updateConfig(conf); err != nil {
		return err
	}
	s.updateConfigChan = make(chan *config.Config, 1)

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
		case newConf := <-s.updateConfigChan:
			err := s.updateConfig(newConf)
			if err != nil {
				s.Log.Error().Str("error", err.Error()).Msg("can't update patterns and filtres")
			}
		case <-s.tomb.Dying():
			return nil
		case diff := <-s.DiffChannel:
			leaks := s.GetLeaks(*diff)
			filtredLeaks := 0
			for _, leak := range leaks {
				if s.filterLeak(leak) {
					filtredLeaks++
					continue
				}
				s.LeakChannel <- &leak
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

func loadPatternsFromFile(file string) ([]patternType, error) {
	rawPatterns := []config.Pattern{}
	rawData, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("can't read file '%s' with: %v", file, err)
	}
	if err := yaml.Unmarshal(rawData, &rawPatterns); err != nil {
		return nil, fmt.Errorf("can't parse file '%s' with: %v", file, err)
	}
	result, err := compilePatterns(rawPatterns)
	if err != nil {
		return nil, fmt.Errorf("can't compile file '%s' with: %v", file, err)
	}
	return result, nil
}

func loadPatternsFromPath(path string) ([]patternType, error) {
	result := []patternType{}
	files, err := filepath.Glob(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		patterns, err := loadPatternsFromFile(file)
		if err != nil {
			return nil, err
		}
		result = append(result, patterns...)
	}
	return result, nil
}

func (s *Searcher) updateConfig(conf *config.Config) error {
	newCompiledPatterns, err := compilePatterns(conf.Patterns)
	if err != nil {
		return err
	}
	newCompiledFiltres, err := compilePatterns(conf.Filters)
	if err != nil {
		return err
	}

	if conf.Common.PatternsPath != "" {
		newFilePatterns, err := loadPatternsFromPath(conf.Common.PatternsPath)
		if err != nil {
			return err
		}
		newCompiledPatterns = append(newCompiledPatterns, newFilePatterns...)
	}

	if conf.Common.FiltresPath != "" {
		newFileFilters, err := loadPatternsFromPath(conf.Common.FiltresPath)
		if err != nil {
			return err
		}
		newCompiledFiltres = append(newCompiledFiltres, newFileFilters...)
	}
	s.patterns, s.filters = newCompiledPatterns, newCompiledFiltres
	s.Log.Info().Int("patterns", len(newCompiledPatterns)).Int("filters", len(newCompiledFiltres)).Msg("loaded")
	s.config = conf
	return nil
}

func (s *Searcher) Status(repoURL string) RepoStats {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()
	if repoStats, ok := s.stats[repoURL]; ok {
		return repoStats
	}
	return RepoStats{}
}

func (s *Searcher) GetLeaks(diff hungryfox.Diff) []hungryfox.Leak {
	leaks := make([]hungryfox.Leak, 0)
	lines := strings.Split(diff.Content, "\n")
	for _, line := range lines {
		for _, pattern := range s.patterns {
			repoFilePath := fmt.Sprintf("%s/%s", diff.RepoURL, diff.FilePath)
			if !pattern.FileRe.MatchString(repoFilePath) {
				continue
			}
			if pattern.ContentRe.MatchString(line) {
				if len(line) > 1024 {
					line = line[:1024]
				}
				leak := hungryfox.Leak{
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
				}
				leaks = append(leaks, leak)
			}
		}
	}
	return leaks
}

func (s *Searcher) filterLeak(leak hungryfox.Leak) bool {
	for _, filter := range s.filters {
		if filter.FileRe.MatchString(fmt.Sprintf("%s/%s", leak.RepoURL, leak.FilePath)) && filter.ContentRe.MatchString(leak.LeakString) {
			return true
		}
	}
	return false
}
