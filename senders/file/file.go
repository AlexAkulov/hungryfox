package file

import (
	"encoding/json"
	"os"

	"github.com/AlexAkulov/hungryfox"
)

type File struct {
	LeaksFile string
	DepsFile  string
}

func (self *File) Start() error {
	return nil
}

func (self *File) Stop() error {
	return nil
}

func (self *File) Send(item interface{}) error {
	switch msg := item.(type) {
	case hungryfox.Leak:
		appendLine(self.LeaksFile, msg)
	case hungryfox.VulnerableDependency:
		appendLine(self.DepsFile, msg)
	}
	return nil
}

func appendLine(file string, item interface{}) error {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, _ := json.Marshal(item)
	f.Write(line)
	f.WriteString("\n")
	return nil
}
