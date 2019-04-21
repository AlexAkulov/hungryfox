package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/AlexAkulov/hungryfox/helpers"
	yaml "gopkg.in/yaml.v2"
)

type Vault struct {
	Enable       bool     `yaml:"enable"`
	VaultURL     string   `yaml:"vault_url"`
	RoleID       string   `yaml:"role_id"`
	SecretID     string   `yaml:"secret_id"`
	Token        string   `yaml:"token"`
	Paths        []string `yaml:"paths"`
	ExcludePaths []string `yaml:"exclude_paths"`
}

type SMTP struct {
	Enable       bool   `yaml:"enable"`
	From         string `yaml:"mail_from"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	TLS          bool   `yaml:"tls"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Recipient    string `yaml:"recipient"`
	SentToAuthor bool   `yaml:"sent_to_author"`
	Delay        string `yaml:"delay"`
}

type Config struct {
	Common   *Common   `yaml:"common"`
	Inspect  []Inspect `yaml:"inspect"`
	Patterns []Pattern `yaml:"patterns"`
	Filters  []Pattern `yaml:"filters"`
	SMTP     *SMTP     `yaml:"smtp"`
	Vault    *Vault    `yaml:"vault"`
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
	PatternsPath           string `yaml:"patterns_path"`
	FiltersPath            string `yaml:"filters_path"`
	Workers                int    `yaml:"workers"`
	HistoryPastLimit       time.Time
	ScanInterval           time.Duration
}

type Pattern struct {
	Name    string `yaml:"name"`
	File    string `yaml:"file"`
	Content string `yaml:"content"`
}

func defaultConfig() *Config {
	return &Config{
		SMTP: &SMTP{
			Delay: "5m",
		},
	}
}

func LoadConfig(configLocation string) (*Config, error) {
	config := defaultConfig()
	configYaml, err := ioutil.ReadFile(configLocation)
	if err != nil {
		return nil, fmt.Errorf("can't read with: %v", err)
	}
	err = yaml.Unmarshal(configYaml, &config)
	if err != nil {
		return nil, fmt.Errorf("can't parse with: %v", err)
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
