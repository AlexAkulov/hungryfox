package matching

import (
	"fmt"
	"regexp"
)

var matchAllRegex = regexp.MustCompile(".+")

func CompileRegex(pattern string) *regexp.Regexp {
	if pattern == "*" || pattern == "" {
		return matchAllRegex
	}
	if regex, err := regexp.Compile(pattern); err != nil {
		panic(fmt.Errorf("can't compile pattern file regexp '%s' with: %v", pattern, err))
	} else {
		return regex
	}
}
