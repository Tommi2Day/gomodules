package maillib

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/go-mail"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type MailConfigType struct {
	Server      string
	Port        int
	Username    string
	Password    string
	Tls         bool
	Maxsize     int64
	ContentType mail.ContentType
}

// mailConfig holds Mails Server Config
var mailConfig = MailConfigType{
	Server:      "127.0.0.1",
	Port:        25,
	Username:    "",
	Password:    "",
	Tls:         false,
	Maxsize:     5120000,
	ContentType: mail.TypeTextPlain,
}

var files []string = nil
var sendCC []string = nil
var sendBcc []string = nil
var sendFrom = ""

// SetConfig set Mail server parameter
func SetConfig(server string, port int, username string, password string, tls bool) {
	if len(server) > 0 {
		mailConfig.Server = server
	}
	if port > 0 {
		mailConfig.Port = port
	}
	if len(username) > 0 && len(password) > 0 {
		mailConfig.Username = username
		mailConfig.Password = password
	}
	if tls {
		mailConfig.Tls = tls
	}
	files = nil
	sendCC = nil
	sendBcc = nil
}

// Cc sets the list of comma seperated CC'ed recipents
func Cc(cclist string) {
	sendCC = strings.Split(strings.TrimSpace(cclist), ",")
}

// Bcc sets the list of comma seperated Bcc'ed recipents
func Bcc(bcclist string) {
	sendBcc = strings.Split(strings.TrimSpace(bcclist), ",")
}

// From changes the From Mail header
func From(from string) {
	sendFrom = from
}

// Attach ads list of files (comma seperated full path)
func Attach(filelist string) {
	files = strings.Split(strings.TrimSpace(filelist), ",")
}

// SetMaxSize limits the size of attached files, 0 to disable
func SetMaxSize(maxsize int64) {
	mailConfig.Maxsize = maxsize
}

// SetContentType allows to modify the Content type of the mail
func SetContentType(contentType mail.ContentType) {
	mailConfig.ContentType = contentType
}

// GetConfig returns current Mail conf
func GetConfig() MailConfigType {
	return mailConfig
}

// SendMail send a mail with the given content
func SendMail(to string, subject string, text string) (err error) {
	var errtxt string
	var sendTo []string = nil
	var c *mail.Client
	sendTo = strings.Split(strings.TrimSpace(to), ",")
	if sendTo == nil {
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
	m.SetBodyString(mailConfig.ContentType, text)

	// handle Attachments
	if len(files) > 0 {
		maxsize := mailConfig.Maxsize
		for _, fn := range files {

			if maxsize == 0 {
				m.AttachFile(fn, mail.WithFileName(filepath.Base(fn)))
			} else {
				f, err := os.Open(fn)
				if err != nil {
					errtxt = fmt.Sprintf("Cannot read %s: %v", fn, err)
					err = errors.New(errtxt)
					log.Error(errtxt)
					return
				}
				lr := io.LimitReader(f, maxsize)
				m.AttachReader(fn, lr, mail.WithFileName(filepath.Base(fn)))
				_ = f.Close()
			}
			log.Debugf("Mail: Attach %s", fn)
		}
	}

	// create mail client
	c, err = mail.NewClient(mailConfig.Server, mail.WithPort(mailConfig.Port))
	if err != nil {
		errtxt = fmt.Sprintf("failed to create mail client: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}
	if mailConfig.Tls {
		c.SetSSL(mailConfig.Tls)
		log.Debug("Mail: Use SSL")
	}
	if len(mailConfig.Username) > 0 && len(mailConfig.Password) > 0 {
		c.SetUsername(mailConfig.Username)
		c.SetPassword(mailConfig.Password)
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
