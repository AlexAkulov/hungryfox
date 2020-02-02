package searcher

import (
	"fmt"
	. "github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/helpers"
	"github.com/AlexAkulov/hungryfox/searcher/leaks"
	"github.com/AlexAkulov/hungryfox/searcher/matching"
	"github.com/AlexAkulov/hungryfox/searcher/stats"
	"github.com/AlexAkulov/hungryfox/searcher/vulnerabilities"
	"github.com/go-kit/kit/metrics"
	"github.com/rs/zerolog"
	"gopkg.in/tomb.v2"
)

type Metrics struct {
	Leaks           metrics.Counter
	Vulnerabilities metrics.Counter
}

type SearcherDispatcher struct {
	Workers                int //TODO: separate workers config
	DiffChannel            <-chan *Diff
	LeakChannel            chan<- *Leak
	VulnerabilitiesChannel chan<- *VulnerableDependency
	Log                    zerolog.Logger
	Metrics                Metrics

	updateConfigChan chan<- *config.Config
	updateStatsChan  chan interface{}
	config           *config.Config
	stats            map[string]stats.RepoStats
	leakMatchers     *leaks.Matchers
	suppressions     *[]matching.Suppression
	tomb             tomb.Tomb
}

func (d *SearcherDispatcher) Update(conf *config.Config) {
	d.tomb.Go(func() error { return d.updateConfig(conf) })
}

func (d *SearcherDispatcher) Start(conf *config.Config) error {
	if err := d.updateConfig(conf); err != nil {
		return err
	}
	d.updateConfigChan = make(chan *config.Config, 1)

	if d.Workers < 1 {
		return fmt.Errorf("workers count can't be less than 1")
	}

	d.stats = map[string]stats.RepoStats{}
	d.updateStatsChan = make(chan interface{})
	d.tomb.Go(d.statsUpdaterWorker)

	var (
		leaksDiffChannel, depsDiffChannel <-chan (*Diff)
		depsChannel                       chan *Dependency
	)
	if d.config.Common.EnableExposuresScanner {
		leaksDiffChannel, depsDiffChannel = helpers.Duplicate(d.DiffChannel, 200)
		depsChannel = make(chan *Dependency)
	} else {
		leaksDiffChannel = d.DiffChannel
	}

	for i := 0; i < d.Workers; i++ {
		if d.config.Common.EnableLeaksScanner {
			leaksWorker := d.makeLeakWorker(leaksDiffChannel)
			d.tomb.Go(leaksWorker.Run)
		}
		if d.config.Common.EnableExposuresScanner {
			depsWorker := d.makeDepsWorker(depsDiffChannel, depsChannel)
			d.tomb.Go(depsWorker.Run)
			vulnsWorker := d.makeVulnsWorker(depsChannel, d.VulnerabilitiesChannel)
			d.tomb.Go(vulnsWorker.Run)
		}
	}
	return nil
}

func (d *SearcherDispatcher) Status(repoURL string) stats.RepoStats {
	if repoStats, ok := d.stats[repoURL]; ok {
		return repoStats
	}
	return stats.RepoStats{}
}

func (d *SearcherDispatcher) Stop() error {
	d.tomb.Kill(nil)
	return d.tomb.Wait()
}

func (d *SearcherDispatcher) makeLeakWorker(diffChannel <-chan *Diff) *Worker {
	return &Worker{
		Searcher: &leaks.LeakSearcher{
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

func (d *SearcherDispatcher) makeDepsWorker(diffChannel <-chan *Diff, depsChannel chan<- *Dependency) *Worker {
	return &Worker{
		Searcher: &vulnerabilities.DepsSearcher{
			DepsChannel: depsChannel,
			Log:         d.Log,
		},
		DiffChannel: diffChannel,
		Log:         d.Log,
		Dying:       d.tomb.Dying(),
	}
}

func (d *SearcherDispatcher) makeVulnsWorker(depsChannel <-chan *Dependency, vulnsChannel chan<- *VulnerableDependency) *vulnerabilities.VulnerabilitiesWorker {
	ossCreds := vulnerabilities.Credentials{
		User:     d.config.Exposures.OssIndexUser,
		Password: d.config.Exposures.OssIndexPassword,
	}
	return &vulnerabilities.VulnerabilitiesWorker{
		Searcher:    vulnerabilities.NewVulnsSearcher(vulnsChannel, d.Log, ossCreds, d.suppressions),
		DepsChannel: depsChannel,
		Log:         d.Log,
		Dying:       d.tomb.Dying(),
	}
}

func (d *SearcherDispatcher) updateConfig(conf *config.Config) error {
	newCompiledPatterns, err := matching.CompilePatterns(conf.Patterns)
	if err != nil {
		return err
	}
	newCompiledFiltres, err := matching.CompilePatterns(conf.Filters)
	if err != nil {
		return err
	}
	newSuppressions := []matching.Suppression{}

	if conf.Common.PatternsPath != "" {
		newFilePatterns, err := matching.LoadPatternsFromPath(conf.Common.PatternsPath)
		if err != nil {
			return err
		}
		newCompiledPatterns = append(newCompiledPatterns, newFilePatterns...)
	}

	if conf.Common.FiltresPath != "" {
		newFileFilters, err := matching.LoadPatternsFromPath(conf.Common.FiltresPath)
		if err != nil {
			return err
		}
		newCompiledFiltres = append(newCompiledFiltres, newFileFilters...)
	}

	if conf.Exposures.SuppressionsPath != "" {
		newSuppressions, err = matching.LoadSuppressionsFromPath(conf.Exposures.SuppressionsPath)
		if err != nil {
			return err
		}
	}

	matchers := leaks.Matchers{Patterns: newCompiledPatterns, Filters: newCompiledFiltres}
	d.leakMatchers = &matchers
	d.Log.Info().Int("patterns", len(newCompiledPatterns)).Int("filters", len(newCompiledFiltres)).Msg("loaded")
	d.suppressions = &newSuppressions
	d.Log.Info().Int("suppressions", len(newSuppressions)).Msg("loaded")
	d.config = conf
	return nil
}

func (d *SearcherDispatcher) statsUpdaterWorker() (err error) {
	defer helpers.RecoverTo(&err)
	for {
		someDiff := <-d.updateStatsChan
		switch diff := someDiff.(type) {
		case stats.LeakStatsDiff:
			{
				stats := d.stats[diff.RepoURL]
				stats.LeaksFiltered += diff.Filtered
				stats.LeaksFound += diff.Found
				d.stats[diff.RepoURL] = stats
				d.Metrics.Leaks.Add(float64(diff.Found))
			}
		case stats.VulnerabilityStatsDiff:
			{
				stats := d.stats[diff.RepoURL]
				stats.VulnerabilitiesFound += diff.Found
				stats.VulnerabilitiesSuppressed += diff.Suppressed
				d.stats[diff.RepoURL] = stats
				d.Metrics.Vulnerabilities.Add(float64(diff.Found))
			}
		}
	}
}
