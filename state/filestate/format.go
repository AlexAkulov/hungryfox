package filestate

import "time"

type RepoJSON struct {
	RepoURL    string   `yaml:"url"`
	RepoPath   string   `yaml:"repo_path"`
	DataPath   string   `yaml:"data_path"`
	Refs       []string `yaml:"refs"`
	ScanStatus ScanJSON `yaml:"scan_status"`
}

type ScanJSON struct {
	StartTime time.Time `yaml:"start_time"`
	EndTime   time.Time `yaml:"end_time"`
	Success   bool      `yaml:"success"`
}
