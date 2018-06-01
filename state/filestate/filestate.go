package filestate

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/AlexAkulov/hungryfox"

	"gopkg.in/tomb.v2"
	"gopkg.in/yaml.v2"
)

type StateManager struct {
	Location string
	sync     sync.RWMutex
	state    map[hungryfox.RepoID]*hungryfox.RepoState
	tomb     tomb.Tomb
}

func (s *StateManager) Start() error {
	if err := s.load(); err != nil {
		return err
	}

	s.tomb.Go(func() error {
		saveTicker := time.NewTicker(time.Minute)
		for {
			select {
			case <-s.tomb.Dying():
				return s.save()
			case <-saveTicker.C:
				if err := s.save(); err != nil {
					fmt.Printf("can't save state with err: %v\n", err)
				}
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
	s.state = map[hungryfox.RepoID]*hungryfox.RepoState{}
	stateYaml, err := ioutil.ReadFile(s.Location)
	if err != nil {
		return fmt.Errorf("can't open, %v", err)
	}
	s.sync.Lock()
	defer s.sync.Unlock()
	err = yaml.Unmarshal([]byte(stateYaml), s.state)
	if err != nil {
		return fmt.Errorf("can't parse, %v", err)
	}
	return nil
}

func (s *StateManager) save() error {
	if _, err := os.Stat(s.Location); os.IsNotExist(err) {
		if _, err := os.Create(s.Location); err != nil {
			return fmt.Errorf("can't create, %v", err)
		}
	}
	s.sync.Lock()
	defer s.sync.Unlock()
	stateContent, err := yaml.Marshal(s.state)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(s.Location, stateContent, 0644); err != nil {
		return fmt.Errorf("can't save, %v", err)
	}
	return nil
}

func (s StateManager) SetState(repoID hungryfox.RepoID, state hungryfox.RepoState) {
	s.sync.Lock()
	defer s.sync.Unlock()
	s.state[repoID] = &state
}

func (s StateManager) GetState(id hungryfox.RepoID) hungryfox.RepoState {
	s.sync.RLock()
	defer s.sync.RUnlock()
	if state, ok := s.state[id]; ok {
		return *state
	}
	return hungryfox.RepoState{}
}
