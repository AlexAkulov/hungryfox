package scanmanager

import (
	"time"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/helpers"
	"github.com/AlexAkulov/hungryfox/hercules"
	"github.com/AlexAkulov/hungryfox/repolist"

	"github.com/rs/zerolog"
	"gopkg.in/tomb.v2"
)

// ScanManager -
type ScanManager struct {
	DiffChannel  chan<- *hungryfox.Diff
	Log          zerolog.Logger
	StateManager hungryfox.IStateManager

	config      *config.Config
	tomb        tomb.Tomb
	currentRepo int
	repoList    *repolist.RepoList
}

// SetConfig - update configuration
func (sm *ScanManager) SetConfig(config *config.Config) {
	sm.config = config
	sm.Log.Debug().Str("service", "scan manager").Msg("config reloaded")
	sm.updateScanList()
}

// Status - get status for current repo
func (sm *ScanManager) Status() *hungryfox.Repo {
	return sm.repoList.GetRepoByIndex(sm.currentRepo)
}

func (sm *ScanManager) updateScanList() {
	sm.Log.Debug().Str("status", "start").Msg("update scan list")
	if sm.repoList == nil {
		sm.repoList = &repolist.RepoList{State: sm.StateManager}
	}
	sm.repoList.Clear()
	for _, inspectObject := range sm.config.Inspect {
		switch inspectObject.Type {
		case "path":
			sm.inspectRepoPath(inspectObject)
		case "github":
			sm.inspectGithub(inspectObject)
		default:
			sm.Log.Error().Str("type", inspectObject.Type).Msg("unsupported type")
		}
	}
	sm.Log.Debug().Str("status", "complete").Msg("update scan list")
}

// Start - start ScanManager instance
func (sm *ScanManager) Start(config *config.Config) error {
	sm.config = config
	sm.currentRepo = -1
	sm.updateScanList()

	sm.tomb.Go(func() error {
		updateTicker := time.NewTicker(time.Minute * 30)
		scanTimer := time.NewTimer(time.Second)
		for {
			select {
			case <-sm.tomb.Dying():
				return nil
			case <-updateTicker.C:
				sm.updateScanList()
			case <-scanTimer.C:
				scanTimer = sm.scanNext()
			}
		}
	})
	return nil
}

func (sm *ScanManager) Stop() error {
	sm.tomb.Kill(nil)
	if err := sm.tomb.Wait(); err != nil {
		sm.Log.Error().Str("error", err.Error()).Msg("stop")
	}
	return nil
}

func (sm *ScanManager) scanNext() *time.Timer {
	rID := sm.repoList.GetRepoForScan()
	if rID < 0 {
		waitTime := time.Duration(time.Minute)
		sm.Log.Debug().Str("wait", helpers.PrettyDuration(waitTime)).Msg("no repo for scan")
		return time.NewTimer(waitTime)
	}
	sm.currentRepo = rID
	defer func() {
		sm.currentRepo = -1
	}()
	r := sm.repoList.GetRepoByIndex(rID)
	elapsedTime := time.Since(r.Scan.EndTime)
	if elapsedTime > sm.config.Common.ScanInterval {
		sm.Log.Info().Str("data_path", r.Location.DataPath).Str("repo_path", r.Location.RepoPath).Msg("start scan")
		sm.ScanRepo(rID)
		return time.NewTimer(0)
	}
	waitTime := sm.config.Common.ScanInterval - elapsedTime
	sm.Log.Info().Str("wait", helpers.PrettyDuration(waitTime)).Msg("wait repo for scan")
	return time.NewTimer(waitTime)
}

// ScanRepo - open exist git repository and fing leaks
func (sm *ScanManager) ScanRepo(index int) {
	r := sm.repoList.GetRepoByIndex(index)
	if r == nil {
		panic("bad index")
	}
	sm.Log.Debug().Str("repo_url", r.Location.URL).Int("refs", len(r.State.Refs)).Msg("state loaded")
	r.Repo = &repo.Repo{
		DiffChannel:      sm.DiffChannel,
		HistoryPastLimit: sm.config.Common.HistoryPastLimit,
		DataPath:         r.Location.DataPath,
		RepoPath:         r.Location.RepoPath,
		URL:              r.Location.URL,
		CloneURL:         r.Location.CloneURL,
		AllowUpdate:      r.Options.AllowUpdate,
	}
	r.Repo.SetRefs(r.State.Refs)
	startScan := time.Now().UTC()
	r.Scan.StartTime = startScan
	sm.repoList.UpdateRepo(*r)

	err := openScanClose(*r)
	r.State.Refs = r.Repo.GetRefs()
	newR := hungryfox.Repo{
		Location: r.Location,
		Options:  r.Options,
		State:    hungryfox.RepoState{Refs: r.Repo.GetRefs()},
		Scan: hungryfox.ScanStatus{
			StartTime: startScan,
			EndTime:   time.Now().UTC(),
			Success:   err == nil,
		},
	}
	sm.repoList.UpdateRepo(newR)

	if err != nil {
		sm.Log.Error().Str("data_path", newR.Location.DataPath).Str("repo_path", newR.Location.RepoPath).Str("error", err.Error()).Msg("scan failed")
	} else {
		sm.Log.Info().Str("data_path", newR.Location.DataPath).Str("repo_path", newR.Location.RepoPath).Str("duration", helpers.PrettyDuration(time.Since(newR.Scan.StartTime))).Msg("scan completed")
	}
	return
}

func openScanClose(r hungryfox.Repo) error {
	if err := r.Repo.Open(); err != nil {
		return err
	}
	defer r.Repo.Close()
	return r.Repo.Scan()
}
