package hungryfox

import (
	"time"

	"github.com/package-url/packageurl-go"
)

type Diff struct {
	CommitHash  string
	RepoURL     string
	RepoPath    string
	FilePath    string
	LineBegin   int
	Content     string
	AuthorEmail string
	Author      string
	TimeStamp   time.Time
}

type RepoOptions struct {
	AllowUpdate bool
}

type RepoLocation struct {
	CloneURL string
	URL      string
	DataPath string
	RepoPath string
}

type RepoState struct {
	Refs []string
}

type ScanStatus struct {
	StartTime time.Time
	EndTime   time.Time
	Success   bool
}

type Repo struct {
	Options  RepoOptions
	Location RepoLocation
	State    RepoState
	Scan     ScanStatus
	Repo     IRepo
}

type Dependency struct {
	Purl packageurl.PackageURL
	Diff
}

type IMessageSender interface {
	Start() error
	Accepts(interface{}) bool
	Send(interface{}) error
	Stop() error
}

type IRepo interface {
	Open() error
	Close() error
	Scan() error
	GetProgress() int
	GetRefs() []string
	SetRefs([]string)
}

type IStateManager interface {
	Load(string) (RepoState, ScanStatus)
	Save(Repo)
}

type IVulnerabilitySearcher interface {
	Search([]Dependency) error
}

type Leak struct {
	PatternName  string    `json:"pattern_name"`
	Regexp       string    `json:"pattern"`
	FilePath     string    `json:"filepath"`
	RepoPath     string    `json:"repo_path"`
	LeakString   string    `json:"leak"`
	RepoURL      string    `json:"repo_url"`
	CommitHash   string    `json:"commit"`
	TimeStamp    time.Time `json:"ts"`
	Line         int       `json:"line"`
	CommitAuthor string    `json:"author"`
	CommitEmail  string    `json:"email"`
}

type VulnerableDependency struct {
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`

	DependencyName string    `json:"dep_name"`
	Version        string    `json:"dep_version"`
	FilePath       string    `json:"filepath"`
	RepoPath       string    `json:"repo_path"`
	RepoURL        string    `json:"repo_url"`
	CommitHash     string    `json:"commit"`
	TimeStamp      time.Time `json:"ts"`
	CommitAuthor   string    `json:"author"`
	CommitEmail    string    `json:"email"`
}

type Vulnerability struct {
	Source        string   `json:"source"`
	Id            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	CvssScore     float32  `json:"cvssScore"`
	CvssVector    string   `json:"cvssVector"`
	Cwe           string   `json:"cwe"`
	Cve           string   `json:"cve"`
	Reference     string   `json:"reference"`
	VersionRanges []string `json:"versionRanges"`
}
