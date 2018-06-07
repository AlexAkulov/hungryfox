package filestate

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/AlexAkulov/hungryfox"

	"gopkg.in/tomb.v2"
	"gopkg.in/yaml.v2"
)

type StateManager struct {
	Location            string
	state               map[string]hungryfox.Repo
	tomb                tomb.Tomb
	saveRepoChan        chan hungryfox.Repo
	loadRepoChan        chan hungryfox.Repo
	loadRepoChanRequest chan string
}

func (s *StateManager) Start() error {
	if err := s.load(); err != nil {
		return err
	}
	s.saveRepoChan = make(chan hungryfox.Repo)
	s.loadRepoChan = make(chan hungryfox.Repo)
	s.loadRepoChanRequest = make(chan string)

	s.tomb.Go(func() error {
		saveTicker := time.NewTicker(time.Minute)
		for {
			select {
			case <-s.tomb.Dying():
				return s.saveToFile()
			case <-saveTicker.C:
				if err := s.saveToFile(); err != nil {
					fmt.Printf("can't save state with err: %v\n", err)
				}
			case r := <-s.saveRepoChan:
				s.state[r.Location.URL] = r
			case url := <-s.loadRepoChanRequest:
				if r, ok := s.state[url]; ok {
					s.loadRepoChan <- r
					continue
				}
				s.loadRepoChan <- hungryfox.Repo{}
			}
		}
	})
	return nil
}

func (s *StateManager) Stop() error {
	s.tomb.Kill(nil)
	return s.tomb.Wait()
}

func (s *StateManager) load() error {
	if _, err := os.Stat(s.Location); os.IsNotExist(err) {
		if _, err := os.Create(s.Location); err != nil {
			return fmt.Errorf("can't create with: %v", err)
		}
	}

	stateRaw, err := ioutil.ReadFile(s.Location)
	if err != nil {
		return fmt.Errorf("can't open, %v", err)
	}
	if s.state, err = converFromRawData(stateRaw); err != nil {
		return fmt.Errorf("can't parse, %v", err)
	}
	return nil
}

func convertToRawData(stateStruct map[string]hungryfox.Repo) ([]byte, error) {
	fileStruct := []RepoJSON{}
	for _, r := range stateStruct {
		fileStruct = append(fileStruct, RepoJSON{
			RepoURL:  r.Location.URL,
			CloneURL: r.Location.CloneURL,
			RepoPath: r.Location.RepoPath,
			DataPath: r.Location.DataPath,
			Refs:     r.State.Refs,
			ScanStatus: ScanJSON{
				StartTime: r.Scan.StartTime,
				EndTime:   r.Scan.EndTime,
				Success:   r.Scan.Success,
			},
		})
	}
	return yaml.Marshal(&fileStruct)
}

func converFromRawData(rawData []byte) (map[string]hungryfox.Repo, error) {
	stateJSON := []RepoJSON{}
	if err := yaml.Unmarshal(rawData, &stateJSON); err != nil {
		return nil, err
	}
	result := map[string]hungryfox.Repo{}
	for _, r := range stateJSON {
		result[r.RepoURL] = hungryfox.Repo{
			Location: hungryfox.RepoLocation{
				URL:      r.RepoURL,
				CloneURL: r.CloneURL,
				DataPath: r.DataPath,
				RepoPath: r.RepoPath,
			},
			State: hungryfox.RepoState{
				Refs: r.Refs,
			},
			Scan: hungryfox.ScanStatus{
				StartTime: r.ScanStatus.StartTime,
				EndTime:   r.ScanStatus.EndTime,
				Success:   r.ScanStatus.Success,
			},
		}
	}
	return result, nil
}

func (s *StateManager) saveToFile() error {
	if _, err := os.Stat(s.Location); os.IsNotExist(err) {
		if _, err := os.Create(s.Location); err != nil {
			return fmt.Errorf("can't create, %v", err)
		}
	}
	rawData, err := convertToRawData(s.state)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(s.Location, rawData, 0644); err != nil {
		return fmt.Errorf("can't save, %v", err)
	}
	return nil
}

func (s StateManager) Save(r hungryfox.Repo) {
	s.saveRepoChan <- r
}

func (s StateManager) Load(url string) (hungryfox.RepoState, hungryfox.ScanStatus) {
	s.loadRepoChanRequest <- url
	r := <-s.loadRepoChan
	return r.State, r.Scan
}
