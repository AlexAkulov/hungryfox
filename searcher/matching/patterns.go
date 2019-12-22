package matching

import (
	"fmt"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/helpers"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

type PatternType struct {
	Name      string
	ContentRe *regexp.Regexp
	FileRe    *regexp.Regexp
	Entropies *EntropyType
}

type EntropyType struct {
	WordMin float64
	LineMin float64
}

func LoadPatternsFromPath(path string) ([]PatternType, error) {
	result := []PatternType{}
	files, err := filepath.Glob(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		patterns, err := loadPatternsFromFile(file)
		if err != nil {
			return nil, err
		}
		result = append(result, patterns...)
	}
	return result, nil
}

func loadPatternsFromFile(file string) ([]PatternType, error) {
	rawPatterns := []config.Pattern{}
	rawData, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("can't read file '%s' with: %v", file, err)
	}
	if err := yaml.Unmarshal(rawData, &rawPatterns); err != nil {
		return nil, fmt.Errorf("can't parse file '%s' with: %v", file, err)
	}
	result, err := CompilePatterns(rawPatterns)
	if err != nil {
		return nil, fmt.Errorf("can't compile file '%s' with: %v", file, err)
	}
	return result, nil
}

func CompilePatterns(configPatterns []config.Pattern) (result []PatternType, err error) {
	defer helpers.RecoverTo(&err)

	for _, configPattern := range configPatterns {
		p := PatternType{
			Name:      configPattern.Name,
			FileRe:    CompileRegex(configPattern.File),
			ContentRe: CompileRegex(configPattern.Content),
		}
		if configPattern.Entropies != nil {
			p.Entropies = &EntropyType{
				WordMin: configPattern.Entropies.WordMin,
				LineMin: configPattern.Entropies.LineMin,
			}
		}
		result = append(result, p)
	}
	return result, nil
}
