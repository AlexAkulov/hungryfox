package email

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"github.com/AlexAkulov/hungryfox"
	"github.com/facebookgo/muster"
	"github.com/rs/zerolog"
	"gopkg.in/gomail.v2"
)

type Kind int

const (
	Leaks Kind = iota
	Exposures
)

// Config - SMTP settings
type Config struct {
	From           string
	SMTPHost       string
	SMTPPort       int
	InsecureTLS    bool
	Username       string
	Password       string
	Delay          time.Duration
	SendToAuthor   bool
	RecipientRegex string
}

// Sender - send email
type Sender struct {
	Kind
	AuditorEmail string
	Config       *Config
	Log          zerolog.Logger

	recipientRegex *regexp.Regexp
	template       *template.Template
	muster         *muster.Client
}

func (s *Sender) Accepts(item interface{}) bool {
	switch item.(type) {
	case hungryfox.Leak:
		return s.Kind == Leaks
	case hungryfox.VulnerableDependency:
		return s.Kind == Exposures
	default:
		return false
	}
}

// Send - send leaks
func (s *Sender) Send(item interface{}) error {
	s.muster.Work <- item
	return nil
}

// Start - start sender
func (s *Sender) Start() error {
	t, err := smtp.Dial(fmt.Sprintf("%s:%d", s.Config.SMTPHost, s.Config.SMTPPort))
	if err != nil {
		return err
	}
	defer t.Close()
	// Test TLS handshake
	if err := t.StartTLS(&tls.Config{
		InsecureSkipVerify: s.Config.InsecureTLS,
		ServerName:         s.Config.SMTPHost,
	}); err != nil {
		return err
	}
	// Test authentication
	if s.Config.Password != "" {
		if err := t.Auth(smtp.PlainAuth(
			"",
			s.Config.Username,
			s.Config.Password,
			s.Config.SMTPHost,
		)); err != nil {
			return err
		}
	}
	if s.template, err = s.getTemplate(); err != nil {
		return err
	}
	var batchMaker func() muster.Batch
	if s.Kind == Leaks {
		batchMaker = s.leakBatchMaker
	} else {
		batchMaker = s.exposuresBatchMaker
	}
	s.muster = &muster.Client{
		MaxBatchSize:         100,
		MaxConcurrentBatches: 1,
		BatchTimeout:         s.Config.Delay,
		BatchMaker:           batchMaker,
	}
	return s.muster.Start()
}

func (s *Sender) sendMessage(recipient string, subject string, messageData interface{}) error {
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

	m.SetHeader("Subject", subject)
	m.AddAlternativeWriter("text/html", func(w io.Writer) error {
		return s.template.Execute(w, messageData)
	})
	return d.DialAndSend(m)
}

func (s *Sender) isOkRecipient(recipient string) bool {
	if s.recipientRegex == nil {
		if s.Config.RecipientRegex == "" {
			return true
		}
		var err error
		s.recipientRegex, err = regexp.Compile(s.Config.RecipientRegex)
		if err != nil {
			s.Log.Warn().Err(err).Msg("Invalid regex in recipient_filter configuration")
			return true
		}
	}
	return s.recipientRegex.MatchString(recipient)
}

func (s *Sender) getTemplate() (*template.Template, error) {
	if s.Kind == Leaks {
		return template.New("leaksmail").Parse(leaksTemplate)
	} else {
		return template.New("exposuresmail").Parse(exposuresTemplate)
	}
}

// Stop - stop sender
func (s *Sender) Stop() error {
	return s.muster.Stop()
}
