package maillib

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/go-mail"
	"path/filepath"
)

type MailConfigType struct {
	Server      string
	Port        int
	Username    string
	Password    string
	Tls         bool
	Maxsize     int
	ContentType mail.ContentType
}

// MailConfig holds Mails Server Config
var MailConfig = MailConfigType{
	Server:      "127.0.0.1",
	Port:        25,
	Username:    "",
	Password:    "",
	Tls:         false,
	Maxsize:     5120000,
	ContentType: mail.TypeTextPlain,
}

// SetMailConfig set Mail server parameter
func SetMailConfig(server string, port int, username string, password string, tls bool) {
	if len(server) > 0 {
		MailConfig.Server = server
	}
	if port > 0 {
		MailConfig.Port = port
	}
	if len(username) > 0 && len(password) > 0 {
		MailConfig.Username = username
		MailConfig.Password = password
	}
	if tls {
		MailConfig.Tls = tls
	}
}

// SendMail send a mail with the given content
func SendMail(sendFrom string, sendTo []string, subject string, text string, files []string,
	sendCC []string, sendBcc []string) (err error) {
	var errtxt string
	var c *mail.Client
	if len(sendTo) == 0 {
		errtxt = "Cannot send Mail without email address"
		err = errors.New(errtxt)
		log.Errorf(errtxt)
		return
	}

	// create message
	m := mail.NewMsg()
	m.SetDate()

	// set from address
	if len(sendFrom) > 0 {
		if err = m.From(sendFrom); err != nil {
			errtxt = fmt.Sprintf("failed to set From address: %s", sendFrom)
			log.Warn(errtxt)
		}
		_ = m.ReplyTo(sendFrom)
	}
	// add recipients
	for _, to := range sendTo {
		if err = m.AddTo(to); err != nil {
			errtxt = fmt.Sprintf("failed to set To address: %s", to)
			err = errors.New(errtxt)
			log.Errorf(errtxt)
			return
		}
	}
	if len(sendCC) > 0 {
		for _, cc := range sendCC {
			if err = m.AddCc(cc); err != nil {
				errtxt = fmt.Sprintf("failed to set CC address: %s", cc)
				err = errors.New(errtxt)
				log.Errorf(errtxt)
				return
			}
		}
	}
	if len(sendBcc) > 0 {
		for _, bcc := range sendBcc {
			if err = m.AddBcc(bcc); err != nil {
				errtxt = fmt.Sprintf("failed to set bcc address: %s", bcc)
				err = errors.New(errtxt)
				log.Errorf(errtxt)
				return
			}
		}
	}
	log.Debug("Mail: Recipients added")

	// set content
	m.Subject(subject)
	m.SetBodyString(MailConfig.ContentType, text)

	// handle Attachments
	if len(files) > 0 {
		for _, fn := range files {
			// ToDo: Limit Filesize
			m.AttachFile(fn, mail.WithFileName(filepath.Base(fn)))
			log.Debugf("Mail: Attach %s", fn)
		}
	}

	// create mail client
	c, err = mail.NewClient(MailConfig.Server, mail.WithPort(MailConfig.Port))
	if err != nil {
		errtxt = fmt.Sprintf("failed to create mail client: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}
	if MailConfig.Tls {
		c.SetSSL(MailConfig.Tls)
		log.Debug("Mail: Use SSL")
	}
	if len(MailConfig.Username) > 0 && len(MailConfig.Password) > 0 {
		c.SetUsername(MailConfig.Username)
		c.SetPassword(MailConfig.Password)
		c.SetSMTPAuth(mail.SMTPAuthPlain)
		log.Debug("Mail: Use Authentication")
	}
	if err = c.DialAndSend(m); err != nil {
		errtxt = fmt.Sprintf("failed to send mail: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}
	return
}
