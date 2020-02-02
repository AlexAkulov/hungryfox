package matching

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	. "github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/helpers"
	"gopkg.in/yaml.v2"
)

type Suppression struct {
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
	FilePath       string `yaml:"file_path"`

	Source string `yaml:"source"`
	Id     string `yaml:"id"`
	Title  string `yaml:"title"`
	Cve    string `yaml:"cve"`
}

func FilterSuppressed(dep *Dependency, vulns []Vulnerability, suppressions []Suppression) []Vulnerability {
	for _, s := range suppressions {
		vulns = s.Filter(dep, vulns)
	}
	return vulns
}

func (s *Suppression) IsMatch(dep *Dependency, vulnerability *Vulnerability) bool {
	if !s.Repository.MatchString(dep.Diff.RepoURL) ||
		!s.DependencyName.MatchString(dep.Purl.Name) ||
		!s.Version.MatchString(dep.Purl.Version) ||
		!s.FilePath.MatchString(dep.Diff.FilePath) {
		return false
	}
	return s.shouldSuppress(vulnerability)
}

func (s *Suppression) Filter(dep *Dependency, vulns []Vulnerability) []Vulnerability {
	if !s.Repository.MatchString(dep.Diff.RepoURL) ||
		!s.DependencyName.MatchString(dep.Purl.Name) ||
		!s.Version.MatchString(dep.Purl.Version) ||
		!s.FilePath.MatchString(dep.Diff.FilePath) {
		return vulns
	}
	filtered := make([]Vulnerability, 0, len(vulns))
	for _, v := range vulns {
		if !s.shouldSuppress(&v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func (s *Suppression) shouldSuppress(v *Vulnerability) bool {
	return s.Source.MatchString(v.Source) &&
		s.Id.MatchString(v.Id) &&
		s.Title.MatchString(v.Title) &&
		s.Cve.MatchString(v.Cve)
}

func LoadSuppressionsFromPath(path string) ([]Suppression, error) {
	results := []Suppression{}
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

func loadSuppressionsFromFile(file string) (s []Suppression, err error) {
	defer helpers.RecoverTo(&err)

	rawSuppressions := []suppressionDto{}
	rawData, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("can't read file '%s' with: %v", file, err)
	}
	if err := yaml.Unmarshal(rawData, &rawSuppressions); err != nil {
		return nil, fmt.Errorf("can't parse file '%s' with: %v", file, err)
	}

	return compileSuppressions(rawSuppressions), nil
}

func compileSuppressions(rawSuppressions []suppressionDto) []Suppression {
	suppressions := make([]Suppression, len(rawSuppressions))
	for i, rawSup := range rawSuppressions {
		suppressions[i] = Suppression{
			Repository:     CompileRegex(rawSup.Repository),
			DependencyName: CompileRegex(rawSup.DependencyName),
			Version:        CompileRegex(rawSup.Version),
			FilePath:       CompileRegex(rawSup.FilePath),
			Source:         CompileRegex(rawSup.Source),
			Id:             CompileRegex(rawSup.Id),
			Title:          CompileRegex(rawSup.Title),
			Cve:            CompileRegex(rawSup.Cve),
		}
	}
	return suppressions
}
