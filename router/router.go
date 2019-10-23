package router

import (
	"fmt"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/helpers"
	"github.com/AlexAkulov/hungryfox/senders/email"
	"github.com/AlexAkulov/hungryfox/senders/file"
	"github.com/AlexAkulov/hungryfox/senders/webhook"

	"github.com/rs/zerolog"
	"gopkg.in/tomb.v2"
)

type LeaksRouter struct {
	LeakChannel <-chan *hungryfox.Leak
	Config      *config.Config
	Log         zerolog.Logger

	senders map[string]hungryfox.IMessageSender
	tomb    tomb.Tomb
}

func (r *LeaksRouter) Start() error {
	delay, err := helpers.ParseDuration(r.Config.SMTP.Delay)
	if err != nil {
		return fmt.Errorf("can't parse delay with: %v", err)
	}
	r.senders = map[string]hungryfox.IMessageSender{}
	if r.Config.SMTP.Enable {
		r.senders["email"] = &email.Sender{
			AuditorEmail: r.Config.SMTP.Recipient,
			Config: &email.Config{
				From:        r.Config.SMTP.From,
				SMTPHost:    r.Config.SMTP.Host,
				SMTPPort:    r.Config.SMTP.Port,
				InsecureTLS: !r.Config.SMTP.TLS,
				Username:    r.Config.SMTP.Username,
				Password:    r.Config.SMTP.Password,
				Delay:       delay,
			},
			Log: r.Log,
		}
	}

	if r.Config.WebHook.Enable {
		r.senders["webhook"] = &webhook.Sender{
			Method:  r.Config.WebHook.Method,
			URL:     r.Config.WebHook.URL,
			Headers: r.Config.WebHook.Headers,
		}
	}

	r.senders["file"] = &file.File{
		LeaksFile: r.Config.Common.LeaksFile,
	}

	for senderName, sender := range r.senders {
		if err := sender.Start(); err != nil {
			return err
		}
		r.Log.Debug().Str("service", senderName).Msg("strated")
	}

	r.tomb.Go(func() error {
		for {
			select {
			case <-r.tomb.Dying(): // Stop
				return nil
			case leak := <-r.LeakChannel:
				for _, sender := range r.senders {
					sender.Send(*leak)
				}
			}
		}
	})
	return nil
}

func (r *LeaksRouter) Stop() error {
	r.tomb.Kill(nil)
	r.tomb.Wait()
	for _, sender := range r.senders {
		sender.Stop()
	}
	return nil
}
