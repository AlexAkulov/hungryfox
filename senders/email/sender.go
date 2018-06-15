package email

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"
	"strings"

	"github.com/AlexAkulov/hungryfox"

	"github.com/facebookgo/muster"
	"gopkg.in/gomail.v2"
)

type mailTemplateStruct struct {
	LeaksCount int
	FilesCount int
	Repos      []*mailTemplateRepoStruct
}

type mailTemplateRepoStruct struct {
	RepoURL string
	Items   []hungryfox.Leak
}

func (s *Sender) batchMaker() muster.Batch {
	return &batch{
		Sender: s,
		Repos:  map[string]*mailTemplateRepoStruct{},
		Files:  map[string]struct{}{},
	}
}

type batch struct {
	LeaksCount int
	Repos      map[string]*mailTemplateRepoStruct
	Files      map[string]struct{}
	Sender     *Sender
}

func (b *batch) Fire(notifier muster.Notifier) {
	defer notifier.Done()
	if b.LeaksCount < 1 {
		return
	}
	messageData := &mailTemplateStruct{
		FilesCount: len(b.Files),
		LeaksCount: b.LeaksCount,
	}
	for _, repo := range b.Repos {
		messageData.Repos = append(messageData.Repos, repo)
	}
	err := b.Sender.sendMessage(b.Sender.AuditorEmail, messageData)
	if err != nil {
		b.Sender.Log.Error().Str("error", err.Error()).Msg("can't send email")
	}
}

func (b *batch) Add(item interface{}) {
	leak := item.(hungryfox.Leak)
	leak.LeakString = strings.TrimSpace(leak.LeakString)
	if len(leak.LeakString) > 512 {
		leak.LeakString = "too long"
	}
	if b.Repos[leak.RepoURL] == nil {
		b.Repos[leak.RepoURL] = &mailTemplateRepoStruct{
			RepoURL: leak.RepoURL,
			Items:   []hungryfox.Leak{},
		}
	}
	b.Repos[leak.RepoURL].Items = append(b.Repos[leak.RepoURL].Items, leak)
	b.Files[fmt.Sprintf("%s/%s", leak.RepoURL, leak.FilePath)] = struct{}{}
	b.LeaksCount++
}

func (s *Sender) sendMessage(recipient string, messageData *mailTemplateStruct) error {
	d := gomail.Dialer{
		Host: s.Config.SMTPHost,
		Port: s.Config.SMTPPort,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: s.Config.InsecureTLS,
			ServerName:         s.Config.SMTPHost,
		},
	}
	if s.Config.Password != "" {
		d.Auth = smtp.PlainAuth(
			"",
			s.Config.Username,
			s.Config.Password,
			s.Config.SMTPHost)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.Config.From)
	m.SetHeader("To", strings.Split(recipient, ",")...)

	var subject string
	if len(messageData.Repos) == 1 {
		subject = fmt.Sprintf("Found %d leaks in %s", messageData.LeaksCount, messageData.Repos[0].RepoURL)
	} else {
		subject = fmt.Sprintf("Found %d leaks in %d repos", messageData.LeaksCount, len(messageData.Repos))
	}
	m.SetHeader("Subject", subject)
	m.AddAlternativeWriter("text/html", func(w io.Writer) error {
		return s.template.Execute(w, messageData)
	})
	return d.DialAndSend(m)
}
