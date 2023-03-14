// Package maillib collect functions for sending mails
package maillib

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wneessen/go-mail"
)

// MailConfigType stuct for config properties
type MailConfigType struct {
	Server      string
	Port        int
	Username    string
	Password    string
	Maxsize     int64
	ContentType mail.ContentType
	SSLinsecure bool
	SSL         bool
	StartTLS    bool
	Timeout     time.Duration
}

// mailConfig holds Mails Server Config
var mailConfig MailConfigType

var files []string
var sendTo []string
var sendCC []string
var sendBcc []string
var sendFrom = ""
var tlsConfig *tls.Config

// EnableSSL allows usage SMTPS Connections (e.g. Port 465)
func EnableSSL(insecure bool) {
	tlsConfig = &tls.Config{
		//nolint gosec
		InsecureSkipVerify: true,
	}
	mailConfig.StartTLS = false
	mailConfig.SSLinsecure = insecure
	mailConfig.SSL = true
}

// EnableTLS allows usage of STARTTLS (e.g. Port 587)
func EnableTLS(insecure bool) {
	tlsConfig = &tls.Config{
		//nolint gosec
		InsecureSkipVerify: true,
	}
	mailConfig.StartTLS = true
	mailConfig.SSLinsecure = insecure
	mailConfig.SSL = false
}

// SetConfig set Mail server parameter
func SetConfig(server string, port int, username string, password string) {
	mailConfig.Server = server
	mailConfig.Port = port
	mailConfig.Username = username
	mailConfig.Password = password
	mailConfig.StartTLS = false
	mailConfig.SSLinsecure = false
	mailConfig.SSL = false
	mailConfig.Timeout = 15 * time.Second

	files = nil
	sendCC = nil
	sendBcc = nil
	mailConfig.Maxsize = 0
	mailConfig.ContentType = mail.TypeTextPlain
}

// Cc sets the list of comma delimited CC'ed recipents
func Cc(cclist string) {
	sendCC = strings.Split(strings.TrimSpace(cclist), ",")
}

// Bcc sets the list of comma delimited Bcc'ed recipents
func Bcc(bcclist string) {
	sendBcc = strings.Split(strings.TrimSpace(bcclist), ",")
}

// Attach adds list of files (comma delimited full path)
func Attach(filelist []string) {
	files = filelist
}

// SetMaxSize limits the size of attached files, 0 to disable
func SetMaxSize(maxsize int64) {
	mailConfig.Maxsize = maxsize
}

// SetContentType allows to modify the Content type of the tests
func SetContentType(contentType mail.ContentType) {
	mailConfig.ContentType = contentType
}

// SetTimeout configure max time to connect
func SetTimeout(seconds uint) {
	timeout := time.Second * time.Duration(seconds)
	mailConfig.Timeout = timeout
}

// GetConfig returns current Mail conf
func GetConfig() MailConfigType {
	return mailConfig
}

// buildRecipients add recipients to tests
func buildRecipients(m *mail.Msg) (err error) {
	var errtxt string
	// add recipients
	for _, r := range sendTo {
		if err = m.AddTo(r); err != nil {
			errtxt = fmt.Sprintf("failed to set To address %s:%v", r, err)
			err = errors.New(errtxt)
			log.Errorf(errtxt)
			return
		}
	}
	if len(sendCC) > 0 {
		for _, cc := range sendCC {
			if err = m.AddCc(cc); err != nil {
				errtxt = fmt.Sprintf("failed to set CC address%s:%v", cc, err)
				err = errors.New(errtxt)
				log.Errorf(errtxt)
				return
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
	return
}

// SendMail send a mail with the given content
func SendMail(from string, to string, subject string, text string) (err error) {
	var errtxt string
	var c *mail.Client
	sendTo = strings.Split(strings.TrimSpace(to), ",")
	sendFrom = strings.TrimSpace(from)
	if sendTo == nil {
		errtxt = "cannot send Mail without email address"
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

	if err = buildRecipients(m); err != nil {
		return
	}
	// set content
	m.Subject(subject)
	m.SetBodyString(mailConfig.ContentType, text)

	// handle Attachments
	if len(files) > 0 {
		err = attachFiles(m)
		if err != nil {
			errtxt = fmt.Sprintf("failed to attach a file: %s", err)
			err = errors.New(errtxt)
			log.Error(errtxt)
			return
		}
	}

	// create mail client
	c, err = mail.NewClient(mailConfig.Server, mail.WithPort(mailConfig.Port), mail.WithTimeout(mailConfig.Timeout))
	if err != nil {
		errtxt = fmt.Sprintf("failed to create tests client: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}

	c.SetSSL(mailConfig.SSL)
	c.SetTLSPolicy(mail.NoTLS)
	if mailConfig.SSLinsecure {
		_ = c.SetTLSConfig(tlsConfig)
	}
	if mailConfig.StartTLS {
		c.SetTLSPolicy(mail.TLSMandatory)
	}
	log.Debug("Mail: Use SSL")

	if len(mailConfig.Username) > 0 && len(mailConfig.Password) > 0 {
		c.SetUsername(mailConfig.Username)
		c.SetPassword(mailConfig.Password)
		c.SetSMTPAuth(mail.SMTPAuthPlain)
		log.Debug("Mail: Use Authentication")
	}
	if err = c.DialAndSend(m); err != nil {
		errtxt = fmt.Sprintf("failed to send tests: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}
	return
}

func attachFiles(m *mail.Msg) error {
	maxsize := mailConfig.Maxsize
	if maxsize > 0 {
		log.Debugf("Mail: File Limit %d", maxsize)
	}
	for _, fn := range files {
		log.Debugf("Attach %s", fn)
		if maxsize == 0 {
			m.AttachFile(fn, mail.WithFileName(filepath.Base(fn)))
		} else {
			//nolint gosec
			f, oserr := os.Open(fn)
			if oserr != nil {
				errtxt := fmt.Sprintf("Cannot read %s: %v", fn, oserr)
				err := errors.New(errtxt)
				log.Error(errtxt)
				return err
			}
			lr := io.LimitReader(f, maxsize)
			m.AttachReader(fn, lr, mail.WithFileName(filepath.Base(fn)))
			_ = f.Close()
		}
	}
	return nil
}
