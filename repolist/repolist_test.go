package repolist

import (
	"testing"
	"time"

	"github.com/AlexAkulov/hungryfox"
	. "github.com/smartystreets/goconvey/convey"
)

type FakeStateManager struct {
	data []hungryfox.Repo
}

func (f FakeStateManager) Load(url string) (hungryfox.RepoState, hungryfox.ScanStatus) {
	for i := range f.data {
		if f.data[i].Location.URL == url {
			return f.data[i].State, f.data[i].Scan
		}
	}
	return hungryfox.RepoState{}, hungryfox.ScanStatus{}
}

func (f FakeStateManager) Save(r hungryfox.Repo) {
	for i := range f.data {
		if f.data[i].Location.URL == r.Location.URL {
			f.data[i] = r
			return
		}
	}
	f.data = append(f.data, r)
}

func TestGetRepoForScan(t *testing.T) {
	Convey("addRepoToScan", t, func() {
		now := time.Now().UTC()
		testData := []hungryfox.Repo{
			hungryfox.Repo{
				Location: hungryfox.RepoLocation{URL: "minute"},
				Scan: hungryfox.ScanStatus{
					StartTime: now.Add(-time.Minute),
					EndTime:   now.Add(-time.Minute),
				},
			},
			hungryfox.Repo{
				Location: hungryfox.RepoLocation{URL: "day"},
				Scan: hungryfox.ScanStatus{
					StartTime: now.Add(-time.Hour * 24),
					EndTime:   now.Add(-time.Hour * 24),
				},
			},
			hungryfox.Repo{
				Location: hungryfox.RepoLocation{URL: "no scan"},
				Scan: hungryfox.ScanStatus{
					EndTime: now,
				},
			},
			hungryfox.Repo{
				Location: hungryfox.RepoLocation{URL: "hour"},
				Scan: hungryfox.ScanStatus{
					StartTime: now.Add(-time.Hour),
					EndTime:   now.Add(-time.Hour),
				},
			},
		}
		stateManager := FakeStateManager{data: testData}
		rl := RepoList{State: stateManager}
		for _, r := range testData {
			rl.AddRepo(r)
		}

		So(len(rl.list), ShouldEqual, 4)
		for _, expectedID := range []int{2, 1, 3, 0} {
			id := rl.GetRepoForScan()
			So(id, ShouldEqual, expectedID)
			r := *rl.GetRepoByIndex(id)
			So(r, ShouldResemble, testData[expectedID])
			r.Scan.StartTime = now
			r.Scan.EndTime = now
			rl.UpdateRepo(r)
		}
	})
}
