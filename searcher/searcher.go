package searcher

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/vault"
	"github.com/rs/zerolog"
	sync "github.com/sasha-s/go-deadlock"
	"gopkg.in/tomb.v2"
	yaml "gopkg.in/yaml.v2"
)

var matchAllRegex = regexp.MustCompile(".+")

type rePatternType struct {
	Name      string
	ContentRe *regexp.Regexp
	FileRe    *regexp.Regexp
}

type vaultFilterType struct {
	SecretPath *regexp.Regexp
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

	vault 			*vault.Vault
	config           *config.Config
	stats            map[string]RepoStats
	statsMutex       sync.RWMutex
	tomb             tomb.Tomb
	rePatterns       []rePatternType
	vaultPattrens    map[string]string
	filters          []rePatternType
	updateConfigChan chan *config.Config
}

func compilePatterns(configPatterns []config.Pattern) ([]rePatternType, error) {
	result := make([]rePatternType, 0)
	for _, configPattern := range configPatterns {
		p := rePatternType{
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

	if conf.Vault.Enable {
		s.vault = &vault.Vault{Config: conf.Vault}
		if err := s.vault.Start(); err != nil {
			return err
		}
	}

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
			for i := range leaks {
				if s.filterLeak(leaks[i]) {
					filtredLeaks++
					continue
				}
				s.LeakChannel <- &leaks[i]
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
	if s.vault != nil {
		s.vault.Stop()
	}
	s.tomb.Kill(nil)
	return s.tomb.Wait()
}

func loadPatternsFromFile(file string) ([]rePatternType, error) {
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

func loadPatternsFromPath(path string) ([]rePatternType, error) {
	result := []rePatternType{}
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

	if conf.Common.FiltersPath != "" {
		newFileFilters, err := loadPatternsFromPath(conf.Common.FiltersPath)
		if err != nil {
			return err
		}
		newCompiledFiltres = append(newCompiledFiltres, newFileFilters...)
	}
	s.rePatterns, s.filters = newCompiledPatterns, newCompiledFiltres
	s.Log.Info().Int("patterns", len(newCompiledPatterns)).Int("filters", len(newCompiledFiltres)).Msg("loaded")
	s.config = conf

	if s.config.Vault.Enable {
		secrets, err := s.vault.ReadAll()
		if err != nil {
			return err
		}
		s.vaultPattrens = secrets
	}
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
		// Vault
		for path, secret := range s.vaultPattrens {
			if strings.Contains(line, secret) {
				leaks = append(leaks, hungryfox.Leak{
					RepoPath:     diff.RepoPath,
					FilePath:     diff.FilePath,
					PatternName:  "vault",
					Regexp:       path,
					LeakString:   line,
					CommitHash:   diff.CommitHash,
					TimeStamp:    diff.TimeStamp,
					CommitAuthor: diff.Author,
					CommitEmail:  diff.AuthorEmail,
					RepoURL:      diff.RepoURL,
				})
			}
		}
		// File Patterns
		for _, pattern := range s.rePatterns {
			repoFilePath := fmt.Sprintf("%s/%s", diff.RepoURL, diff.FilePath)
			if !pattern.FileRe.MatchString(repoFilePath) {
				continue
			}
			if pattern.ContentRe.MatchString(line) {
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

func (s *Searcher) filterLeak(leak hungryfox.Leak) bool {
	for _, filter := range s.filters {
		if filter.FileRe.MatchString(fmt.Sprintf("%s/%s", leak.RepoURL, leak.FilePath)) && filter.ContentRe.MatchString(leak.LeakString) {
			return true
		}
	}
	return false
}
