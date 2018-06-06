package scanmanager

import (
	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/hercules"
)

func (sm *ScanManager) DryRun() {
	total := sm.repoList.GetTotalRepos()
	for i := 0; i < total; i++ {
		r := sm.repoList.GetRepoByIndex(i)
		if r == nil {
			panic("bad index")
			return
		}
		if err := sm.getState(r); err != nil {
			sm.Log.Error().Str("error", err.Error()).Str("data_path", r.Location.DataPath).Str("repo_path", r.Location.RepoPath).Msg("can't open repo")
			continue
		}
		sm.Log.Debug().Int("index", i).Int("total", total).Str("data_path", r.Location.DataPath).Str("repo_path", r.Location.RepoPath).Msg("ok")
	}
}

func (sm *ScanManager) getState(r *hungryfox.Repo) error {
	r.Repo = &repo.Repo{
		DiffChannel:      sm.DiffChannel,
		HistoryPastLimit: sm.config.Common.HistoryPastLimit,
		DataPath:         r.Location.DataPath,
		RepoPath:         r.Location.RepoPath,
		URL:              r.Location.URL,
	}
	if err := r.Repo.Open(); err != nil {
		return err
	}
	defer r.Repo.Close()
	r.State.Refs = r.Repo.GetRefs()
	sm.repoList.UpdateRepo(*r)
	return nil
}
