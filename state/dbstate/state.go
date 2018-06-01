package dbstate

import (
	"fmt"
	"strings"

	"github.com/AlexAkulov/hungryfox"

	"upper.io/db.v3/ql"
	"upper.io/db.v3/lib/sqlbuilder"
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

func rIDToString(id hungryfox.RepoID) string {
	return fmt.Sprintf("%s,%s,%s", id.RepoURL, id.DataPath, id.RepoPath)
}

func stringToRID(s string) hungryfox.RepoID {
	a := strings.Split(s, ",")
	return hungryfox.RepoID{
		RepoURL:  a[0],
		DataPath: a[1],
		RepoPath: a[2],
	}
}

func (s StateManager) SetState(repoID hungryfox.RepoID, state hungryfox.RepoState) {
	update := s.db.SelectFrom("state").Where("")

}

func (s StateManager) GetState(id hungryfox.RepoID) hungryfox.RepoState {
	s.db.SelectFrom("state").Where("")
	return hungryfox.RepoState{}
}

func (s *StateManager) setup() error {
	table := `CREATE TABLE state (
	repoid string,
	refs string,
	status string,
	)`
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(table); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
