package vulnerabilities

import (
	ossindex "github.com/A1bemuth/go-oss-index"
	. "github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/searcher/matching"
	"github.com/AlexAkulov/hungryfox/searcher/stats"
	"github.com/rs/zerolog"
)

type exposure struct {
	*Dependency
	*Vulnerability
}

type IDependencyAnalyzer interface {
	Analyze([] Dependency) ([]exposure, error)
}

type VulnerabilitySearcher struct {
	VulnerabilitiesChannel chan<- *VulnerableDependency
	StatsChannel           chan<- interface{}
	Log                    zerolog.Logger

	analyzers    []IDependencyAnalyzer
	suppressions *[]matching.Suppression
}

func NewVulnsSearcher(vulnsChan chan<- *VulnerableDependency, log zerolog.Logger, ossCredentials Credentials, suppressions *[]matching.Suppression) *VulnerabilitySearcher {
	ossAnalyzer := &ossIndexAnalyzer{
		OssIndexClient: ossindex.Client{
			User:     ossCredentials.User,
			Password: ossCredentials.Password,
		},
		Log: log,
	}
	return &VulnerabilitySearcher{
		VulnerabilitiesChannel: vulnsChan,
		Log:                    log,

		analyzers:    []IDependencyAnalyzer{ossAnalyzer},
		suppressions: suppressions,
	}
}

func (s *VulnerabilitySearcher) Search(deps []Dependency) error {
	var exposures []exposure

	for _, a := range s.analyzers {
		result, err := a.Analyze(deps)
		if err != nil {
			return err
		}
		exposures = append(exposures, result...)
	}

	exposures = s.FilterSuppressed(exposures)
	vulnerableDeps := AggregateByDependency(exposures)

	for _, dep := range vulnerableDeps {
		found := len(dep.Vulnerabilities)
		s.Log.Debug().Str("repo", dep.RepoURL).Str("file", dep.FilePath).Int("count", found).Msg("vulnerabilities found")
		s.StatsChannel <- stats.VulnerabilityStatsDiff{
			RepoURL: dep.RepoURL,
			Found:   found,
		}

		s.VulnerabilitiesChannel <- dep
	}

	return nil
}

func (s *VulnerabilitySearcher) FilterSuppressed(exposures []exposure) []exposure {
	if s.suppressions == nil {
		return exposures
	}

	var filtered []exposure
	for _, exp := range exposures {
		for _, supp := range *s.suppressions {
			if supp.IsMatch(exp.Dependency, exp.Vulnerability) {
				s.Log.Info().
					Str("dependency", exp.Dependency.Purl.ToString()).
					Str("vulnerability", exp.Vulnerability.Id).
					Msg("vulnerability suppressed")
				s.StatsChannel <- stats.VulnerabilityStatsDiff{
					RepoURL:    exp.RepoURL,
					Suppressed: 1,
				}
				goto nextExposure
			}
		}
		filtered = append(filtered, exp)
	nextExposure:
	}
	return filtered
}

func AggregateByDependency(exposures []exposure) map[string]*VulnerableDependency {
	vulnerableDeps := make(map[string]*VulnerableDependency)
	for _, exp := range exposures {
		purl := exp.Purl.ToString()
		dep, ok := vulnerableDeps[purl]
		if ok {
			dep.Vulnerabilities = append(dep.Vulnerabilities, *exp.Vulnerability)
		} else {
			vulnerableDeps[purl] = toVulnerableDep(exp.Dependency, []Vulnerability{*exp.Vulnerability})
		}
	}
	return vulnerableDeps
}

func toVulnerableDep(dep *Dependency, vulnerabilities []Vulnerability) *VulnerableDependency {
	return &VulnerableDependency{
		Vulnerabilities: vulnerabilities,
		DependencyName:  dep.Purl.Name,
		Version:         dep.Purl.Version,
		FilePath:        dep.Diff.FilePath,
		RepoPath:        dep.Diff.RepoPath,
		RepoURL:         dep.Diff.RepoURL,
		CommitHash:      dep.Diff.CommitHash,
		TimeStamp:       dep.Diff.TimeStamp,
		CommitAuthor:    dep.Diff.Author,
		CommitEmail:     dep.Diff.AuthorEmail,
	}
}