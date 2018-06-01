package scanmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/helpers"
	"github.com/AlexAkulov/hungryfox/hercules"
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

// InspectRepoPath - open exist git repository and fing leaks
func (sm *ScanManager) InspectRepoPath(id hungryfox.RepoID) error {
	sm.currentRepo = &repo.Repo{
		DiffChannel:      sm.DiffChannel,
		HistoryPastLimit: sm.config.Common.HistoryPastLimit,
		DataPath:         id.DataPath,
		RepoPath:         id.RepoPath,
		URL:              id.RepoURL,
	}
	sm.scanListSync.RLock()
	state := sm.scanList[id]
	sm.scanListSync.RUnlock()
	sm.currentRepo.SetOldRefs(state.Refs)
	err := sm.currentRepo.OpenScanClose()
	newState := sm.currentRepo.Status()
	sm.scanListSync.Lock()
	sm.scanList[id] = newState
	sm.scanListSync.Unlock()
	sm.State.SetState(id, newState)

	if err != nil {
		sm.Log.Error().Str("data_path", id.DataPath).Str("repo_path", id.RepoPath).Str("error", err.Error()).Msg("scan failed")
	} else {
		sm.Log.Info().Str("data_path", id.DataPath).Str("repo_path", id.RepoPath).Int("total", newState.ScanStatus.CommitsTotal).Int("scanned", newState.ScanStatus.CommitsScanned).Str("duration", helpers.PrettyDuration(time.Since(newState.ScanStatus.StartTime))).Msg("scan completed")
	}

	sm.currentRepo = nil
	sm.currentRepoID = hungryfox.RepoID{}
	return err
}

func getRepoID(path string, inspectObject *config.Inspect) hungryfox.RepoID {
	prefix := strings.Replace(inspectObject.TrimPrefix, "\\", "/", -1)
	prefix = strings.TrimSuffix(prefix, "/")
	path = strings.Replace(path, "\\", "/", -1)
	path = strings.TrimPrefix(path, prefix)
	path = strings.Trim(path, "/")
	url := strings.TrimSuffix(inspectObject.URL, "/")
	url = fmt.Sprintf("%s/%s", url, strings.TrimSuffix(path, ".git"))

	return hungryfox.RepoID{
		DataPath: prefix,
		RepoPath: path,
		RepoURL:  url,
	}
}
