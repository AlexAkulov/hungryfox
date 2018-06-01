package scanmanager

import (
	"testing"
	"time"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	. "github.com/smartystreets/goconvey/convey"
)

func TestParseDuration(t *testing.T) {
	Convey("1s", t, func() {
		inspect := &config.Inspect{
			TrimPrefix: "/var/volume/",
			TrimSuffix: ".git",
			URL:        "http://gitlab.com",
		}
		expect := hungryfox.RepoID{
			DataPath: "/var/volume",
			RepoPath: "my/repo.git",
			RepoURL:  "http://gitlab.com/my/repo",
		}
		So(getRepoID("/var/volume/my/repo.git/", inspect), ShouldResemble, expect)
		So(getRepoID("/var/volume/my/repo.git", inspect), ShouldResemble, expect)
	})
}

type fakeState struct {
	state map[hungryfox.RepoID]hungryfox.RepoState
}

func (s fakeState) GetState(id hungryfox.RepoID) hungryfox.RepoState {
	return s.state[id]
}

func (s fakeState) SetState(id hungryfox.RepoID, state hungryfox.RepoState) {
}

func TestGetRepoForScan(t *testing.T) {
	Convey("addRepoToScan", t, func() {
		now := time.Now().UTC()
		fs := fakeState{
			state: map[hungryfox.RepoID]hungryfox.RepoState{
				hungryfox.RepoID{RepoURL: "minute"}: hungryfox.RepoState{
					ScanStatus: hungryfox.ScanStatus{
						StartTime: now.Add(-time.Minute),
						EndTime:   now.Add(-time.Minute),
					},
				},
				hungryfox.RepoID{RepoURL: "day"}: hungryfox.RepoState{
					ScanStatus: hungryfox.ScanStatus{
						StartTime: now.Add(-time.Hour * 24),
						EndTime:   now.Add(-time.Hour * 24),
					},
				},
				hungryfox.RepoID{RepoURL: "no scan"}: hungryfox.RepoState{
					ScanStatus: hungryfox.ScanStatus{
						EndTime: now,
					},
				},
				hungryfox.RepoID{RepoURL: "hour"}: hungryfox.RepoState{
					ScanStatus: hungryfox.ScanStatus{
						StartTime: now.Add(-time.Hour),
						EndTime:   now.Add(-time.Hour),
					},
				},
			},
		}
		sm := ScanManager{State: fs}
		for _, repoURL := range []string{"minute", "day", "no scan", "hour"} {
			sm.addRepoToScan(hungryfox.RepoID{
				RepoURL: repoURL,
			})
		}

		So(len(sm.scanList), ShouldEqual, 4)
		So(sm.scanList, ShouldResemble, fs.state)

		for _, repoURL := range []string{"no scan", "day", "hour", "minute"} {
			id := sm.getRepoForScan()
			repoState := sm.scanList[id]
			So(id.RepoURL, ShouldEqual, repoURL)
			repoState.ScanStatus.StartTime = now
			repoState.ScanStatus.EndTime = now
			sm.scanList[id] = repoState
		}
	})
}
