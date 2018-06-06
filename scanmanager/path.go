package scanmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
)

func expandGlob(inspect *config.Inspect) (map[string]struct{}, error) {
	excludePaths := make(map[string]struct{})
	for _, pattern := range inspect.Paths {
		if !strings.HasPrefix(pattern, "!") {
			continue
		}
		pattern = strings.TrimPrefix(pattern, "!")
		paths, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			excludePaths[path] = struct{}{}
		}
	}
	scanPaths := make(map[string]struct{})
	for _, pattern := range inspect.Paths {
		if strings.HasPrefix(pattern, "!") {
			continue
		}
		paths, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			if _, ok := excludePaths[path]; ok {
				continue
			}
			if f, _ := os.Stat(path); f.IsDir() {
				scanPaths[path] = struct{}{}
			}
		}
	}
	return scanPaths, nil
}

func (sm *ScanManager) inspectRepoPath(inspectObject *config.Inspect) error {
	scanPathList, err := expandGlob(inspectObject)
	if err != nil {
		sm.Log.Error().Str("error", err.Error()).Msg("can't expand glob")
		return err
	}
	for path := range scanPathList {
		location := getRepoLocation(path, inspectObject)
		sm.repoList.AddRepo(hungryfox.Repo{
			Options:  hungryfox.RepoOptions{AllowUpdate: false},
			Location: location,
		})
	}
	return nil
}

func getRepoLocation(path string, inspectObject *config.Inspect) hungryfox.RepoLocation {
	prefix := strings.Replace(inspectObject.TrimPrefix, "\\", "/", -1)
	prefix = strings.TrimSuffix(prefix, "/")
	path = strings.Replace(path, "\\", "/", -1)
	path = strings.TrimPrefix(path, prefix)
	path = strings.Trim(path, "/")
	url := strings.TrimSuffix(inspectObject.URL, "/")
	url = fmt.Sprintf("%s/%s", url, strings.TrimSuffix(path, ".git"))

	return hungryfox.RepoLocation{
		DataPath: prefix,
		RepoPath: path,
		URL:      url,
	}
}
