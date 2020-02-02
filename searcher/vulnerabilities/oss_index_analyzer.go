package vulnerabilities

import (
	ossindex "github.com/A1bemuth/go-oss-index"
	osstypes "github.com/A1bemuth/go-oss-index/types"
	. "github.com/AlexAkulov/hungryfox"
	"github.com/rs/zerolog"
)

type Credentials struct {
	User     string
	Password string
}

type ossIndexAnalyzer struct {
	OssIndexClient ossindex.Client
	Log            zerolog.Logger
}

func (a *ossIndexAnalyzer) Analyze(deps []Dependency) ([]exposure, error) {
	purls, depsMap := mapPurls(deps)
	reports, err := a.OssIndexClient.Get(purls)
	var exposures []exposure
	if err != nil {
		a.Log.Warn().Err(err).Msg("requesting oss index component reports failed")
		return exposures, err
	}

	for _, report := range reports {
		dep, ok := depsMap[report.Coordinates]
		if !ok {
			a.Log.Warn().Str("coordinates", report.Coordinates).Msg("found an oss report but no matching dependency")
			continue
		}

		vulns := getOssVulnerabilities(&report)
		for _, vuln := range vulns {
			exp := exposure{
				Dependency: dep,
				Vulnerability: &vuln,
			}
			exposures = append(exposures, exp)
		}
	}

	return exposures, nil
}

func mapPurls(deps []Dependency) ([]string, map[string]*Dependency) {
	purls := make([]string, len(deps))
	depsMap := make(map[string]*Dependency)
	for i, dep := range deps {
		purl := dep.Purl.ToString()
		purls[i] = purl
		depsMap[purl] = &dep
	}
	return purls, depsMap
}

func getOssVulnerabilities(report *osstypes.ComponentReport) (vulns []Vulnerability) {
	for _, vuln := range report.Vulnerabilities {
		vulns = append(vulns, *toVulnerability(&vuln))
	}
	return vulns
}

func toVulnerability(vuln *osstypes.Vulnerability) *Vulnerability {
	return &Vulnerability{
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