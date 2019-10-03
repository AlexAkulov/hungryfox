package hungryfox

import (
	"time"
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

type IMessageSender interface {
	Start() error
	Send(Leak) error
	Stop() error
}

type IDiffAnalyzer interface {
	Analyze(*Diff)
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
