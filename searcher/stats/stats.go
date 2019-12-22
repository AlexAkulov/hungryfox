package stats

type RepoStats struct {
	LeaksFound                int `json:"leaks_found"`
	LeaksFiltered             int `json:"leaks_filtered"`
	VulnerabilitiesFound      int `json:"vulnerabilities_found"`
	VulnerabilitiesSuppressed int `json:"vulnerabilities_suppressed"`
}

type LeakStatsDiff struct {
	RepoURL  string
	Found    int
	Filtered int
}

type VulnerabilityStatsDiff struct {
	RepoURL    string
	Found      int
	Suppressed int
}