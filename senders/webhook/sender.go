package webhook

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/AlexAkulov/hungryfox"
)

type Sender struct {
	Method  string
	URL     string
	Headers map[string]string
}

func (self *Sender) Start() error {
	return nil
}

func (self *Sender) Stop() error {
	return nil
}

func (self *Sender) Send(leak hungryfox.Leak) error {
	line, _ := json.Marshal(leak)

	req, err := http.NewRequest(self.Method, self.URL, bytes.NewBuffer(line))
	for k, v := range self.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	return err
}
