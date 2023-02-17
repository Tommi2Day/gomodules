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
var mailConfig MailConfigType

var files []string = nil
var sendCC []string = nil
var sendBcc []string = nil
var sendFrom = ""

// SetConfig set Mail server parameter
func SetConfig(server string, port int, username string, password string, tls bool) {
	mailConfig.Server = server
	mailConfig.Port = port
	mailConfig.Username = username
	mailConfig.Password = password
	mailConfig.Tls = tls
	files = nil
	sendCC = nil
	sendBcc = nil
	mailConfig.Maxsize = 0
	mailConfig.ContentType = mail.TypeTextPlain
}

// Cc sets the list of comma seperated CC'ed recipents
func Cc(cclist string) {
	sendCC = strings.Split(strings.TrimSpace(cclist), ",")
}

// Bcc sets the list of comma seperated Bcc'ed recipents
func Bcc(bcclist string) {
	sendBcc = strings.Split(strings.TrimSpace(bcclist), ",")
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
func SendMail(from string, to string, subject string, text string) (err error) {
	var errtxt string
	var sendTo []string
	var c *mail.Client
	sendTo = strings.Split(strings.TrimSpace(to), ",")
	sendFrom = strings.TrimSpace(from)
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
			errtxt = fmt.Sprintf("failed to set From address %s:%v", sendFrom, err)
			log.Warn(errtxt)
		}
		_ = m.ReplyTo(sendFrom)
	}
	// add recipients
	for _, r := range sendTo {
		if err = m.AddTo(r); err != nil {
			errtxt = fmt.Sprintf("failed to set To address %s:%v", r, err)
			err = errors.New(errtxt)
			log.Errorf(errtxt)
		}
	}
	if len(sendCC) > 0 {
		for _, cc := range sendCC {
			if err = m.AddCc(cc); err != nil {
				errtxt = fmt.Sprintf("failed to set CC address%s:%v", cc, err)
				err = errors.New(errtxt)
				log.Errorf(errtxt)
			}
		}
	}
	if len(sendBcc) > 0 {
		for _, bcc := range sendBcc {
			if err = m.AddBcc(bcc); err != nil {
				errtxt = fmt.Sprintf("failed to set bcc address %s:%v", bcc, err)
				err = errors.New(errtxt)
				log.Errorf(errtxt)
				return
			}
		}
	}
	if err != nil {
		return
	}
	log.Debug("Mail: Recipients added")

	// set content
	m.Subject(subject)
	m.SetBodyString(mailConfig.ContentType, text)

	// handle Attachments
	if len(files) > 0 {
		maxsize := mailConfig.Maxsize
		if maxsize > 0 {
			log.Debugf("Mail: File Limit %d", maxsize)
		}
		for _, fn := range files {
			log.Debugf("Attach %s", fn)
			if maxsize == 0 {
				m.AttachFile(fn, mail.WithFileName(filepath.Base(fn)))
			} else {
				f, oserr := os.Open(fn)
				if oserr != nil {
					errtxt = fmt.Sprintf("Cannot read %s: %v", fn, oserr)
					err = errors.New(errtxt)
					log.Error(errtxt)
					return
				} else {
					lr := io.LimitReader(f, maxsize)
					m.AttachReader(fn, lr, mail.WithFileName(filepath.Base(fn)))
					_ = f.Close()
				}

			}

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
