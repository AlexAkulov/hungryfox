package searcher

import "github.com/AlexAkulov/hungryfox/helpers"

type RepoStats struct {
	LeaksFound                int `json:"leaks_found"`
	LeaksFiltred              int `json:"leaks_filtred"`
	VulnerabilitiesFound      int `json:"vulnerabilities_found"`
	VulnerabilitiesSuppressed int `json:"vulnerabilities_suppressed"`
}

type statsKind int

const (
	LeakStat statsKind = iota
	VulnerabilityStat
)

type statsDiff struct {
	Kind     statsKind
	RepoURL  string
	Found    int
	Filtered int
}

func (d *AnalyzerDispatcher) statsUpdaterWorker() (err error) {
	defer helpers.RecoverTo(&err)
	for {
		diff := <-d.updateStatsChan
		stats := d.stats[diff.RepoURL]
		switch diff.Kind {
		case LeakStat:
			{
				stats.LeaksFiltred += diff.Filtered
				stats.LeaksFound += diff.Found
				d.stats[diff.RepoURL] = stats
				d.Metrics.Leaks.Add(float64(diff.Found))
			}
		case VulnerabilityStat:
			{
				stats.VulnerabilitiesFound += diff.Found
				stats.VulnerabilitiesSuppressed += diff.Filtered
				d.stats[diff.RepoURL] = stats
				d.Metrics.Vulnerabilities.Add(float64(diff.Found))
			}
		}
	}
}
