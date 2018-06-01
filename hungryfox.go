package hungryfox

import "time"

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

type RepoID struct {
	RepoURL  string `yaml:"repo_url"`
	DataPath string `yaml:"data_path"`
	RepoPath string `yaml:"repo_path"`
}

func (r RepoID) IsEmpty() bool {
	return r.RepoURL == "" && r.RepoPath == "" && r.DataPath == ""
}

type RepoState struct {
	ScanStatus ScanStatus        `yaml:"scan_status"`
	Refs       map[string]string `yaml:"refs"`
}

type ScanStatus struct {
	StartTime      time.Time `yaml:"start_time"`
	EndTime        time.Time `yaml:"end_time"`
	CommitsTotal   int       `yaml:"commits_total"`
	CommitsScanned int       `yaml:"commits_scaned"`
	Success        bool      `yaml:"success"`
}

type IMessageSender interface {
	Start() error
	Send(*Leak) error
	Stop() error
}

type ILeakSearcher interface {
	Start() error
	SetConfig() error
	Search(*Diff)
	Stop() error
}

type IRepo interface {
	Open() error
	Scan() error
	Status() *ScanStatus
}

type IStateManager interface {
	GetState(RepoID) RepoState
	SetState(RepoID, RepoState)
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
