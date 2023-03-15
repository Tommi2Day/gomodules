// Package maillib collect functions for sending mails
package maillib

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/wneessen/go-mail"
)

// SendMailConfigType Server Config for sending Mails
type SendMailConfigType struct {
	maxSize         int64
	mailContentType mail.ContentType
	serverConfig    *MailConfigType
}

// AddressListType collects all recipients of a mail and attachment
type AddressListType struct {
	files    []string
	sendTo   []string
	sendCC   []string
	sendBcc  []string
	sendFrom string
}

// NewSendMailConfig prepares a new configuration object for sending mails
func NewSendMailConfig(server string, port int, username string, password string) *SendMailConfigType {
	var config SendMailConfigType
	config.serverConfig = NewConfig(server, port, username, password)
	config.maxSize = 5120000
	config.mailContentType = mail.TypeTextPlain
	return &config
}

// NewMail perepare a new Mail Address List
func NewMail(from string, toList string) *AddressListType {
	al := AddressListType{}
	al.To(toList)
	al.sendFrom = from
	return &al
}

// To sets the list of comma delimited recipents
func (l *AddressListType) To(tolist string) {
	l.sendTo = strings.Split(strings.TrimSpace(tolist), ",")
}

// Cc sets the list of comma delimited CC'ed recipents
func (l *AddressListType) Cc(cclist string) {
	l.sendCC = strings.Split(strings.TrimSpace(cclist), ",")
}

// Bcc sets the list of comma delimited Bcc'ed recipents
func (l *AddressListType) Bcc(bcclist string) {
	l.sendBcc = strings.Split(strings.TrimSpace(bcclist), ",")
}

// Attach adds list of files (comma delimited full path)
func (l *AddressListType) Attach(filelist []string) {
	l.files = filelist
}

// SetMaxSize limits the size of attached files, 0 to disable
func (config *SendMailConfigType) SetMaxSize(maxsize int64) {
	config.maxSize = maxsize
}

// SetContentType allows to modify the Content type of the tests
func (config *SendMailConfigType) SetContentType(contentType mail.ContentType) {
	config.mailContentType = contentType
}

// buildRecipients add recipients to tests
func (l *AddressListType) buildRecipients(m *mail.Msg) (c int, err error) {
	var errtxt string
	// add recipients
	for _, r := range l.sendTo {
		if err = m.AddTo(r); err != nil {
			errtxt = fmt.Sprintf("failed to set To address %s:%v", r, err)
			err = errors.New(errtxt)
			log.Errorf(errtxt)
			return
		}
		c++
	}
	if len(l.sendCC) > 0 {
		for _, cc := range l.sendCC {
			if err = m.AddCc(cc); err != nil {
				errtxt = fmt.Sprintf("failed to set CC address%s:%v", cc, err)
				err = errors.New(errtxt)
				log.Errorf(errtxt)
				return
			}
		}
		c++
	}
	if len(l.sendBcc) > 0 {
		for _, bcc := range l.sendBcc {
			if err = m.AddBcc(bcc); err != nil {
				errtxt = fmt.Sprintf("failed to set bcc address %s:%v", bcc, err)
				err = errors.New(errtxt)
				log.Errorf(errtxt)
				return
			}
		}
		c++
	}
	if err != nil {
		return
	}
	log.Debug("Mail: Recipients added")
	return
}

// SendMail send a mail with the given content
func (config *SendMailConfigType) SendMail(addresses *AddressListType, subject string, text string) (err error) {
	var errtxt string
	var c *mail.Client

	// collect addresses
	sendFrom := strings.TrimSpace(addresses.sendFrom)
	if addresses.sendTo == nil {
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

	// build address list
	ac := 0
	if ac, err = addresses.buildRecipients(m); err != nil || ac == 0 {
		return
	}
	// set content
	m.Subject(subject)
	m.SetBodyString(config.mailContentType, text)

	// handle Attachments
	if len(addresses.files) > 0 {
		err = addresses.attachFiles(m, config.maxSize)
		if err != nil {
			errtxt = fmt.Sprintf("failed to attach a file: %s", err)
			err = errors.New(errtxt)
			log.Error(errtxt)
			return
		}
	}

	// create mail client
	c, err = mail.NewClient(config.serverConfig.Server, mail.WithPort(config.serverConfig.Port), mail.WithTimeout(config.serverConfig.Timeout))
	if err != nil {
		errtxt = fmt.Sprintf("failed to create tests client: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}

	// set ssl
	c.SetSSL(config.serverConfig.SSL)
	c.SetTLSPolicy(mail.NoTLS)
	_ = c.SetTLSConfig(config.serverConfig.tlsConfig)
	if config.serverConfig.StartTLS {
		c.SetTLSPolicy(mail.TLSMandatory)
	}

	// login to server if defined
	if len(config.serverConfig.Username) > 0 && len(config.serverConfig.Password) > 0 {
		c.SetUsername(config.serverConfig.Username)
		c.SetPassword(config.serverConfig.Password)
		c.SetSMTPAuth(mail.SMTPAuthPlain)
		log.Debug("Mail: Use Authentication")
	}

	// send mail
	if err = c.DialAndSend(m); err != nil {
		errtxt = fmt.Sprintf("failed to send tests: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}
	return
}

// add files to mail
func (l AddressListType) attachFiles(m *mail.Msg, maxSize int64) error {
	if maxSize > 0 {
		log.Debugf("Mail: File Limit %d", maxSize)
	}
	for _, fn := range l.files {
		log.Debugf("Attach %s", fn)
		if maxSize == 0 {
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
			lr := io.LimitReader(f, maxSize)
			m.AttachReader(fn, lr, mail.WithFileName(filepath.Base(fn)))
		}
	}
	return nil
}

// GetConfig returns current Mail conf
func (config *SendMailConfigType) GetConfig() *SendMailConfigType {
	return config
}
