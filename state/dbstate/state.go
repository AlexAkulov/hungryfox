package dbstate

import (
	"github.com/AlexAkulov/hungryfox"

	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/ql"
)

type StateManager struct {
	Location string
	db       sqlbuilder.Database
}

type State struct {
	RepoID string `db:"repoid"`
	Refs   string `db:"refs"`
	Status string `db:"status"`
}

func (s *StateManager) Start() error {
	settings := ql.ConnectionURL{Database: s.Location}
	var err error
	if s.db, err = ql.Open(settings); err != nil {
		return err
	}
	return nil
}

func (s *StateManager) Stop() error {
	return s.db.Close()
}

func (s StateManager) Save(r hungryfox.Repo) {
}

func (s StateManager) Load(id string) (hungryfox.RepoState, hungryfox.ScanStatus) {
	return hungryfox.RepoState{}, hungryfox.ScanStatus{}
}

func (s *StateManager) setup() error {
	// table := `CREATE TABLE state (
	// repoid string,
	// refs string,
	// status string,
	// )`
	return nil
}
