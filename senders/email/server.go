package email

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"github.com/AlexAkulov/hungryfox"

	"github.com/facebookgo/muster"
	"github.com/rs/zerolog"
)

// Config - SMTP settings
type Config struct {
	From        string
	SMTPHost    string
	SMTPPort    int
	InsecureTLS bool
	Username    string
	Password    string
	Delay       time.Duration
}

// Sender - send email
type Sender struct {
	AuditorEmail string
	Config       *Config
	Log          zerolog.Logger
	template     *template.Template
	muster       *muster.Client
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
	if s.template, err = template.New("mail").Parse(defaultTemplate); err != nil {
		return err
	}
	s.muster = &muster.Client{
		MaxBatchSize:         100,
		MaxConcurrentBatches: 1,
		BatchTimeout:         s.Config.Delay,
		BatchMaker:           s.batchMaker,
	}
	return s.muster.Start()
}

// Stop - stop sender
func (s *Sender) Stop() error {
	return s.muster.Stop()
}

// Send - send leaks
func (s *Sender) Send(leak hungryfox.Leak) error {
	s.muster.Work <- leak
	return nil
}
