package scanmanager

import (
	"github.com/AlexAkulov/hungryfox/hercules"
	"github.com/AlexAkulov/hungryfox"
)

func (sm *ScanManager) DryRun() {
	i := 0
	total := len(sm.scanList)
	for repoID := range sm.scanList {
		i++
		if err := sm.getState(repoID); err != nil {
			sm.Log.Error().Str("error", err.Error()).Str("data_path", repoID.DataPath).Str("repo_path", repoID.RepoPath).Msg("can't open repo")
			continue
		}
		sm.Log.Debug().Int("i", i).Int("t", total).Str("data_path", repoID.DataPath).Str("repo_path", repoID.RepoPath).Msg("ok")
	}
}

func (sm *ScanManager) getState(id hungryfox.RepoID) error {
	repo := &repo.Repo{
		DiffChannel:      sm.DiffChannel,
		HistoryPastLimit: sm.config.Common.HistoryPastLimit,
		DataPath:         id.DataPath,
		RepoPath:         id.RepoPath,
		URL:              id.RepoURL,
	}
	if err := repo.Open(); err != nil {
		return err
	}
	defer repo.Close()
	state := sm.scanList[id]
	state.Refs = repo.GetNewRefs()
	sm.State.SetState(id, state)
	return nil
}
