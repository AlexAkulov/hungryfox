package config

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"time"

	"github.com/AlexAkulov/hungryfox/helpers"

	"gopkg.in/yaml.v2"
)

type SMTP struct {
	Enable       bool   `yaml:"enable"`
	From         string `yaml:"mail_from"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	TLS          bool   `yaml:"tls"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Recipient    string `yaml:"recipient"`
	SentToAuthor bool   `yaml:"sent_to_autor"`
}

type Config struct {
	Common   *Common    `yaml:"common"`
	Inspect  []*Inspect `yaml:"inspect"`
	Patterns []*Pattern `yaml:"patterns"`
	Filters  []*Pattern `yaml:"filters"`
	SMTP     *SMTP      `yaml:"smtp"`
}

type Inspect struct {
	Type       string   `yaml:"type"`
	Paths      []string `yaml:"paths"`
	URL        string   `yaml:"url"`
	Token      string   `yaml:"token"`
	TrimPrefix string   `yaml:"trim_prefix"`
	TrimSuffix string   `yaml:"trim_suffix"`
	WorkDir    string   `yaml:"work_dir"`
	Users      []string `yaml:"users"`
	Repos      []string `yaml:"repos"`
	Orgs       []string `yaml:"orgs"`
}

type Common struct {
	StateFile              string `yaml:"state_file"`
	HistoryPastLimitString string `yaml:"history_limit"`
	LogLevel               string `yaml:"log_level"`
	LeaksFile              string `yaml:"leaks_file"`
	ScanIntervalString     string `yaml:"scan_interval"`
	HistoryPastLimit       time.Time
	ScanInterval           time.Duration
}

type Pattern struct {
	Name      string `yaml:"name"`
	File      string `yaml:"file"`
	Content   string `yaml:"content"`
	ContentRe *regexp.Regexp
	FileRe    *regexp.Regexp
}

func defaultConfig() *Config {
	return &Config{}
}

func compilePatterns(patterns []*Pattern) error {
	for _, pattern := range patterns {
		var err error
		if pattern.File == "*" || pattern.File == "" {
			pattern.FileRe = regexp.MustCompile(".+")
		} else {
			if pattern.FileRe, err = regexp.Compile(pattern.File); err != nil {
				return fmt.Errorf("can't compile pattern file regexp '%s' with: %v", pattern.File, err)
			}
		}
		if pattern.Content == "*" || pattern.Content == "" {
			pattern.ContentRe = regexp.MustCompile(".+")
		} else {
			if pattern.ContentRe, err = regexp.Compile(pattern.Content); err != nil {
				return fmt.Errorf("can't compile pattern content regexp '%s' with: %v", pattern.Content, err)
			}
		}
	}
	return nil
}

func LoadConfig(configLocation string) (*Config, error) {
	config := defaultConfig()
	configYaml, err := ioutil.ReadFile(configLocation)
	if err != nil {
		return nil, fmt.Errorf("can't read with: %v", err)
	}
	err = yaml.Unmarshal([]byte(configYaml), &config)
	if err != nil {
		return nil, fmt.Errorf("can't parse with: %v", err)
	}

	if err := compilePatterns(config.Patterns); err != nil {
		return nil, err
	}
	if err := compilePatterns(config.Filters); err != nil {
		return nil, err
	}
	pastLimit, err := helpers.ParseDuration(config.Common.HistoryPastLimitString)
	if err != nil {
		return nil, err
	}
	config.Common.HistoryPastLimit = time.Now().Add(-pastLimit)
	config.Common.ScanInterval, err = helpers.ParseDuration(config.Common.ScanIntervalString)
	if err != nil {
		return nil, err
	}
	if config.Common.ScanInterval < time.Second {
		return nil, fmt.Errorf("scan_interval so small")
	}
	return config, nil
}

func PrintDefaultConfig() {
	c := defaultConfig()
	d, _ := yaml.Marshal(&c)
	fmt.Print(string(d))
}
