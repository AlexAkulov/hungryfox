package searcher

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	. "github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/helpers"
	"github.com/rs/zerolog"
	"gopkg.in/tomb.v2"
	"gopkg.in/yaml.v2"
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

type statsDiff struct {
	RepoURL  string
	Found    int
	Filtered int
}

type AnalyzerDispatcher struct {
	Workers                int //TODO: separate workers config
	DiffChannel            <-chan *Diff
	LeakChannel            chan<- *Leak
	VulnerabilitiesChannel chan<- *VulnerableDependency
	Log                    zerolog.Logger

	updateConfigChan chan<- *config.Config
	updateStatsChan  chan statsDiff
	config           *config.Config
	stats            map[string]RepoStats
	leakMatchers     *Matchers
	tomb             tomb.Tomb
}

func (d *AnalyzerDispatcher) Update(conf *config.Config) {
	d.tomb.Go(func() error { return d.updateConfig(conf) })
}

func (d *AnalyzerDispatcher) Start(conf *config.Config) error {
	if err := d.updateConfig(conf); err != nil {
		return err
	}
	d.updateConfigChan = make(chan *config.Config, 1)

	if d.Workers < 1 {
		return fmt.Errorf("workers count can't be less than 1")
	}

	d.stats = map[string]RepoStats{}
	d.updateStatsChan = make(chan statsDiff)
	d.tomb.Go(d.updateStatsWorker)

	leaksDiffChannel, depsDiffChannel := helpers.Duplicate(d.DiffChannel, 200)
	depsChannel := make(chan *Dependency)

	for i := 0; i < d.Workers; i++ {
		leaksWorker := d.makeLeakWorker(leaksDiffChannel)
		d.tomb.Go(leaksWorker.Run)
		depsWorker := d.makeDepsWorker(depsDiffChannel, depsChannel)
		d.tomb.Go(depsWorker.Run)
		vulnsWorker := d.makeVulnsWorker(depsChannel, d.VulnerabilitiesChannel)
		d.tomb.Go(vulnsWorker.Run)
	}
	return nil
}

func (d *AnalyzerDispatcher) Status(repoURL string) RepoStats {
	if repoStats, ok := d.stats[repoURL]; ok {
		return repoStats
	}
	return RepoStats{}
}

func (d *AnalyzerDispatcher) Stop() error {
	d.tomb.Kill(nil)
	return d.tomb.Wait()
}

func (d *AnalyzerDispatcher) makeLeakWorker(diffChannel <-chan *Diff) *Worker {
	return &Worker{
		Analyzer: &LeakAnalyzer{
			LeakChannel:  d.LeakChannel,
			Log:          d.Log,
			Matchers:     d.leakMatchers,
			StatsChannel: d.updateStatsChan,
		},
		DiffChannel: diffChannel,
		Log:         d.Log,
		Dying:       d.tomb.Dying(),
	}
}

func (d *AnalyzerDispatcher) makeDepsWorker(diffChannel <-chan *Diff, depsChannel chan<- *Dependency) *Worker {
	return &Worker{
		Analyzer: &DepsAnalyzer{
			DepsChannel: depsChannel,
			Log:         d.Log,
		},
		DiffChannel: diffChannel,
		Log:         d.Log,
		Dying:       d.tomb.Dying(),
	}
}

func (d *AnalyzerDispatcher) makeVulnsWorker(depsChannel <-chan *Dependency, vulnsChannel chan<- *VulnerableDependency) *VulnerabilitiesWorker {
	ossCreds := Credentials{
		User:     d.config.Exposures.OssIndexUser,
		Password: d.config.Exposures.OssIndexPassword,
	}
	return &VulnerabilitiesWorker{
		Searcher:    NewVulnsSearcher(vulnsChannel, d.Log, ossCreds),
		DepsChannel: depsChannel,
		Log:         d.Log,
		Dying:       d.tomb.Dying(),
	}
}

func (d *AnalyzerDispatcher) updateConfig(conf *config.Config) error {
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
	matchers := Matchers{patterns: newCompiledPatterns, filters: newCompiledFiltres}
	d.leakMatchers = &matchers
	d.Log.Info().Int("patterns", len(newCompiledPatterns)).Int("filters", len(newCompiledFiltres)).Msg("loaded")
	d.config = conf
	return nil
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

func (d *AnalyzerDispatcher) updateStatsWorker() (err error) {
	defer func() {
		e := recover()
		if e != nil {
			err = err.(error)
		}
	}()
	for {
		diff := <-d.updateStatsChan
		{
			stats := d.stats[diff.RepoURL]
			stats.LeaksFiltred += diff.Filtered
			stats.LeaksFound += diff.Found
			d.stats[diff.RepoURL] = stats
		}
	}
}
