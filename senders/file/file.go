package file

import (
	"encoding/json"
	"os"

	"github.com/AlexAkulov/hungryfox"
)

type File struct {
	LeaksFile string
}

func (self *File) Start() error {
	return nil
}

func (self *File) Stop() error {
	return nil
}

func (self *File) Send(leak *hungryfox.Leak) error {
	f, err := os.OpenFile(self.LeaksFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, _ := json.Marshal(leak)
	f.Write(line)
	f.WriteString("\n")
	return nil
}
