package searcher

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	. "github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/helpers"
	"gopkg.in/yaml.v2"
)

type suppression struct {
	Repository *regexp.Regexp

	DependencyName *regexp.Regexp
	Version        *regexp.Regexp
	FilePath       *regexp.Regexp

	Source *regexp.Regexp
	Id     *regexp.Regexp
	Title  *regexp.Regexp
	Cve    *regexp.Regexp
}

type suppressionDto struct {
	Repository string `yaml:"repository"`

	DependencyName string `yaml:"dep_name"`
	Version        string `yaml:"dep_version"`
	FilePath       string `yaml:"filepath"`

	Source string `yaml:"source"`
	Id     string `yaml:"id"`
	Title  string `yaml:"title"`
	Cve    string `yaml:"cve"`
}

func filterSuppressed(dep *Dependency, vulns []Vulnerability, suppressions []suppression) []Vulnerability {
	result := make([]Vulnerability, len(vulns))
	copy(result, vulns)
	for _, s := range suppressions {
		s.filter(dep, &result)
	}
	return result
}

func (s *suppression) filter(dep *Dependency, vulns *[]Vulnerability) {
	if !s.Repository.MatchString(dep.Diff.RepoURL) ||
		!s.DependencyName.MatchString(dep.Purl.Name) ||
		!s.Version.MatchString(dep.Purl.Version) ||
		!s.FilePath.MatchString(dep.Diff.FilePath) {
		return
	}
	for i, v := range *vulns {
		if s.shouldSuppress(&v) {
			*vulns = append((*vulns)[:i], (*vulns)[i+1:]...)
		}
	}
}

func (s *suppression) shouldSuppress(v *Vulnerability) bool {
	return s.Source.MatchString(v.Source) &&
		s.Id.MatchString(v.Id) &&
		s.Title.MatchString(v.Title) &&
		s.Cve.MatchString(v.Cve)
}

func loadSuppressionsFromPath(path string) ([]suppression, error) {
	results := []suppression{}
	files, err := filepath.Glob(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		suppressions, err := loadSuppressionsFromFile(file)
		if err != nil {
			return nil, err
		}
		results = append(results, suppressions...)
	}
	return results, nil
}

func loadSuppressionsFromFile(file string) (s []suppression, err error) {
	defer helpers.RecoverTo(&err)

	rawSuppressions := []suppressionDto{}
	rawData, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("can't read file '%s' with: %v", file, err)
	}
	if err := yaml.Unmarshal(rawData, &rawSuppressions); err != nil {
		return nil, fmt.Errorf("can't parse file '%s' with: %v", file, err)
	}

	return nil, nil
}

func compileSuppressions(rawSuppressions []suppressionDto) []suppression {
	suppressions := make([]suppression, len(rawSuppressions))
	for i, rawSup := range rawSuppressions {
		suppressions[i] = suppression{
			Repository:     compileRegex(rawSup.Repository),
			DependencyName: compileRegex(rawSup.DependencyName),
			Version:        compileRegex(rawSup.Version),
			FilePath:       compileRegex(rawSup.FilePath),
			Source:         compileRegex(rawSup.Source),
			Id:             compileRegex(rawSup.Id),
			Title:          compileRegex(rawSup.Title),
			Cve:            compileRegex(rawSup.Cve),
		}
	}
	return suppressions
}

func compileRegex(pattern string) *regexp.Regexp {
	if pattern == "*" || pattern == "" {
		return matchAllRegex
	}
	if regex, err := regexp.Compile(pattern); err != nil {
		panic(fmt.Errorf("can't compile pattern file regexp '%s' with: %v", pattern, err))
	} else {
		return regex
	}
}
