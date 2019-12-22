package vulnerabilities

import (
	ossindex "github.com/A1bemuth/go-oss-index"
	osstypes "github.com/A1bemuth/go-oss-index/types"
	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/searcher/matching"
	"github.com/AlexAkulov/hungryfox/searcher/stats"
	"github.com/rs/zerolog"
)

type Credentials struct {
	User     string
	Password string
}

type VulnerabilitySearcher struct {
	VulnerabilitiesChannel chan<- *hungryfox.VulnerableDependency
	StatsChannel           chan<- interface{}
	Log                    zerolog.Logger

	ossIndexClient ossindex.Client
	suppressions   *[]matching.Suppression
}

func NewVulnsSearcher(vulnsChan chan<- *hungryfox.VulnerableDependency, log zerolog.Logger, ossCredentials Credentials, suppressions *[]matching.Suppression) *VulnerabilitySearcher {
	return &VulnerabilitySearcher{
		VulnerabilitiesChannel: vulnsChan,
		Log:                    log,
		ossIndexClient: ossindex.Client{
			User:     ossCredentials.User,
			Password: ossCredentials.Password,
		},
		suppressions: suppressions,
	}
}

func (s *VulnerabilitySearcher) Search(deps []hungryfox.Dependency) error {
	purls, depsMap := mapPurls(deps)
	reports, err := s.ossIndexClient.Get(purls)
	if err != nil {
		s.Log.Warn().Err(err).Msg("requesting oss index component reports failed")
		return err
	}
	for _, report := range reports {
		dep, ok := depsMap[report.Coordinates]
		if !ok {
			s.Log.Warn().Str("coordinates", report.Coordinates).Msg("found an oss report but no matching dependency")
			continue
		}

		vulns := getOssVulnerabilities(&report)
		found, suppressed := len(vulns), 0
		if len(vulns) == 0 {
			return nil
		}
		s.Log.Debug().Str("repo", dep.RepoURL).Str("file", dep.FilePath).Int("count", found).Msg("vulnerabilities found")

		if s.suppressions != nil {
			vulns = matching.FilterSuppressed(dep, vulns, *s.suppressions)
			suppressed = found - len(vulns)
			if suppressed > 0 {
				s.Log.Debug().Str("repo", dep.RepoURL).Str("file", dep.FilePath).Int("count", suppressed).Msg("vulnerabilities suppressed")
			}
			found = len(vulns)
		}
		if found > 0 {
			s.VulnerabilitiesChannel <- toVulnerableDep(dep, vulns)
		}
		if found > 0 || suppressed > 0 {
			s.StatsChannel <- stats.VulnerabilityStatsDiff{
				RepoURL:  dep.Diff.RepoURL,
				Found:    found,
				Suppressed: suppressed,
			}
		}
	}

	return nil
}

func mapPurls(deps []hungryfox.Dependency) ([]string, map[string]*hungryfox.Dependency) {
	purls := make([]string, len(deps))
	depsMap := make(map[string]*hungryfox.Dependency)
	for i, dep := range deps {
		purl := dep.Purl.ToString()
		purls[i] = purl
		depsMap[purl] = &dep
	}
	return purls, depsMap
}

func toVulnerableDep(dep *hungryfox.Dependency, vulns []hungryfox.Vulnerability) *hungryfox.VulnerableDependency {
	return &hungryfox.VulnerableDependency{
		Vulnerabilities: vulns,
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

func getOssVulnerabilities(report *osstypes.ComponentReport) (vulns []hungryfox.Vulnerability) {
	for _, vuln := range report.Vulnerabilities {
		vulns = append(vulns, *toVulnerability(&vuln))
	}
	return vulns
}

func toVulnerability(vuln *osstypes.Vulnerability) *hungryfox.Vulnerability {
	return &hungryfox.Vulnerability{
		Source:        "Sonatype OSS Index",
		Id:            vuln.Id,
		Title:         vuln.Title,
		Description:   vuln.Description,
		CvssScore:     vuln.CvssScore,
		CvssVector:    vuln.CvssVector,
		Cwe:           vuln.Cwe,
		Cve:           vuln.Cve,
		Reference:     vuln.Reference,
		VersionRanges: vuln.VersionRanges,
	}
}
