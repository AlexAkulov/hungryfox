package scanmanager

import (
	"sync"
	"time"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/helpers"
	"github.com/AlexAkulov/hungryfox/hercules"

	"github.com/rs/zerolog"
	"gopkg.in/tomb.v2"
)

type Scan struct {
	hungryfox.RepoState
	hungryfox.RepoID
	Options struct {
		CloneAndUpdate bool

	}
}

// ScanManager -
type ScanManager struct {
	DiffChannel chan<- *hungryfox.Diff
	Log         zerolog.Logger
	State       hungryfox.IStateManager

	config        *config.Config
	currentRepoID hungryfox.RepoID
	currentRepo   *repo.Repo
	tomb          tomb.Tomb
	scanListSync  sync.RWMutex
	scanList      map[hungryfox.RepoID]hungryfox.RepoState
}

// SetConfig - update configuration
func (sm *ScanManager) SetConfig(config *config.Config) {
	sm.config = config
	sm.Log.Debug().Str("service", "scan manager").Msg("config reloaded")
	sm.updateScanList()
}

func (sm *ScanManager) getRepoForScan() hungryfox.RepoID {
	var rID hungryfox.RepoID
	lastScan := time.Now().UTC()
	sm.scanListSync.RLock()
	defer sm.scanListSync.RUnlock()
	for id, repoState := range sm.scanList {
		if repoState.ScanStatus.StartTime.IsZero() {
			return id
		}
		if repoState.ScanStatus.EndTime.Before(lastScan) {
			rID = id
			lastScan = repoState.ScanStatus.EndTime
		}
	}
	return rID
}

func (sm *ScanManager) addRepoToScan(repoID hungryfox.RepoID) {
	sm.scanListSync.Lock()
	defer sm.scanListSync.Unlock()
	if sm.scanList == nil {
		sm.scanList = make(map[hungryfox.RepoID]hungryfox.RepoState)
	}
	repoState := sm.State.GetState(repoID)
	sm.scanList[repoID] = repoState
}

func (sm *ScanManager) addReposToScan(repoList []hungryfox.RepoID) {
	for _, repoID := range repoList {
		sm.addRepoToScan(repoID)
	}
}

// Status - get status for current repo
func (sm *ScanManager) Status() (*hungryfox.RepoID, *hungryfox.ScanStatus) {
	if !sm.currentRepoID.IsEmpty() {
		s := sm.currentRepo.Status()
		return &sm.currentRepoID, &s.ScanStatus
	}
	return nil, nil
}

func (sm *ScanManager) updateScanList() {
	for _, inspectObject := range sm.config.Inspect {
		switch inspectObject.Type {
		case "path":
			scanPathList, err := expandGlob(inspectObject)
			if err != nil {
				sm.Log.Error().Str("error", err.Error()).Msg("can't expand glob")
				continue
			}
			for path := range scanPathList {
				repoID := getRepoID(path, inspectObject)
				sm.addRepoToScan(repoID)
			}
		// case "github":
		// 	sm.inspectGithub(inspectObject)
		default:
			sm.Log.Error().Str("type", inspectObject.Type).Msg("unsupported type")
		}
	}
}

// Start - start ScanManager instance
func (sm *ScanManager) Start(config *config.Config) error {
	sm.config = config
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
	rID := sm.getRepoForScan()
	if rID.IsEmpty() {
		waitTime := time.Duration(time.Minute)
		sm.Log.Debug().Str("wait", helpers.PrettyDuration(waitTime)).Msg("no repo for scan")
		return time.NewTimer(waitTime)
	}
	sm.currentRepoID = rID
	defer func() {
		sm.currentRepoID = hungryfox.RepoID{}
		sm.currentRepo = nil
	}()
	sm.scanListSync.RLock()
	repoStatus := sm.scanList[rID]
	sm.scanListSync.RUnlock()
	elapsedTime := time.Since(repoStatus.ScanStatus.EndTime)
	if elapsedTime > sm.config.Common.ScanInterval {
		sm.Log.Info().Str("data_path", rID.DataPath).Str("repo_path", rID.RepoPath).Msg("start scan")
		sm.InspectRepoPath(rID)
		return time.NewTimer(0)
	}
	// waitTime := sm.config.Common.ScanInterval - elapsedTime
	waitTime := time.Minute
	sm.Log.Info().Str("wait", helpers.PrettyDuration(waitTime)).Msg("wait repo for scan")
	return time.NewTimer(waitTime)
}

// InspectPath - open exist git repository and fing leaks
// func (this *HungryFox) InspectRepo(repoUrl string, dataPath string) error {

// 	repo := &repo.Repo{
// 		CommitDepth: this.config.Common.CommitsLimit,
// 		DataPath:    dataPath,
// 		URL:         "",
// 		State:       this.State.GetStateByPath(repoUrl),
// 		LS: &LeakSearcher{
// 			Config: this.config,
// 			Log:    this.Log,
// 		},
// 		Log: this.Log,
// 	}

// 	if err := repo.CloneOrUpdate(); err != nil {
// 		return err
// 	}
// 	repo.Scan()
// 	// this.Log.Info().Str("path", repo.RepoPath()).Str("repo_url", repoUrl).Int("count", repo.GetCommitsCount()).Msg("commits scaned")
// 	// leaks := repo.GetLeaks()
// 	// this.Log.Info().Str("path", repo.RepoPath()).Str("repo_url", repoUrl).Int("count", len(leaks)).Msg("leaks found")
// 	// if err := this.SaveLeaks(leaks); err != nil {
// 	// 	this.Log.Error().Str("error", err.Error()).Msg("can't save leaks to file")
// 	// }
// 	this.State.SetState(repo.State)
// 	if err := this.State.Save(); err != nil {
// 		this.Log.Error().Str("error", err.Error()).Msg("can't save state")
// 		return err
// 	}

// 	return nil
// }
