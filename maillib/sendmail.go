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
	ServerConfig    *MailConfigType
}

// NewSendMailConfig prepares a new configuration object for sending mails
func NewSendMailConfig(server string, port int, username string, password string) *SendMailConfigType {
	var config SendMailConfigType
	config.ServerConfig = NewConfig(server, port, username, password)
	config.maxSize = 5120000
	config.mailContentType = mail.TypeTextPlain
	return &config
}

// SetMaxSize limits the size of attached Attachments, 0 to disable
func (config *SendMailConfigType) SetMaxSize(maxsize int64) {
	config.maxSize = maxsize
}

// SetContentType allows to modify the Content type of the test
func (config *SendMailConfigType) SetContentType(contentType mail.ContentType) {
	config.mailContentType = contentType
}

// buildRecipients add recipients to test
func (mt *MailType) buildRecipients(m *mail.Msg) (c int, err error) {
	var errtxt string
	// add recipients
	for _, r := range mt.To {
		if err = m.AddTo(r); err != nil {
			errtxt = fmt.Sprintf("recipients: failed to set SetTo address %s:%v", r, err)
			err = errors.New(errtxt)
			log.Errorf(errtxt)
			return
		}
		c++
	}
	if len(mt.CC) > 0 {
		for _, cc := range mt.CC {
			if err = m.AddCc(cc); err != nil {
				errtxt = fmt.Sprintf("recipients: failed to set CC address%s:%v", cc, err)
				err = errors.New(errtxt)
				log.Errorf(errtxt)
				return
			}
		}
		c++
	}
	if len(mt.Bcc) > 0 {
		for _, bcc := range mt.Bcc {
			if err = m.AddBcc(bcc); err != nil {
				errtxt = fmt.Sprintf("recipients: failed to set bcc address %s:%v", bcc, err)
				err = errors.New(errtxt)
				log.Errorf(errtxt)
				return
			}
		}
		c++
	}
	log.Debugf("recipients: %d Recipients added", c)
	return
}

// SendMail send a mail with the given content
func (config *SendMailConfigType) SendMail(addresses *MailType, subject string, text string) (err error) {
	var errtxt string
	var c *mail.Client

	// collect addresses
	sendFrom := strings.TrimSpace(addresses.From)
	if addresses.To == nil {
		errtxt = "sendmail: cannot send Mail without email address"
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
			errtxt = fmt.Sprintf("sendmail: failed to set From address %s:%v", sendFrom, err)
			log.Warn(errtxt)
		}
		_ = m.ReplyTo(sendFrom)
	}

	// build address list
	ac := 0
	if ac, err = addresses.buildRecipients(m); err != nil || ac == 0 {
		if ac == 0 {
			errtxt = "No recipients given"
		}
		if err != nil {
			errtxt = fmt.Sprintf("build recipients error:%s", err)
		}
		err = errors.New("sendmail:" + errtxt)
		log.Error(errtxt)
		return
	}
	// set content
	m.Subject(subject)
	m.SetBodyString(config.mailContentType, text)
	// handle Attachments
	if len(addresses.Attachments) > 0 {
		err = addresses.attachFiles(m, config.maxSize)
		if err != nil {
			errtxt = fmt.Sprintf("sendmail: failed to attach a file: %s", err)
			err = errors.New(errtxt)
			log.Error(errtxt)
			return
		}
	}
	// prepare mail object
	if c, err = prepareMailObject(config); err != nil {
		errtxt = fmt.Sprintf("sendmail: failed to prepare mail object: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}

	// send mail
	log.Debugf("sendmail: send via %s", c.ServerAddr())
	if err = c.DialAndSend(m); err != nil {
		errtxt = fmt.Sprintf("sendmail: failed: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}
	return
}

func prepareMailObject(config *SendMailConfigType) (c *mail.Client, err error) {
	var errtxt string
	// create mail Conn
	c, err = mail.NewClient(config.ServerConfig.Server,
		mail.WithPort(config.ServerConfig.Port),
		mail.WithTimeout(config.ServerConfig.Timeout),
		mail.WithHELO(config.ServerConfig.HELO))
	if err != nil {
		errtxt = fmt.Sprintf("sendmail: failed to create mail Conn: %s", err)
		err = errors.New(errtxt)
		log.Error(errtxt)
		return
	}

	// set ssl
	c.SetSSL(config.ServerConfig.SSL)
	c.SetTLSPolicy(mail.NoTLS)
	_ = c.SetTLSConfig(config.ServerConfig.tlsConfig)
	if config.ServerConfig.StartTLS {
		c.SetTLSPolicy(mail.TLSMandatory)
		log.Debug("sendmail: Use StartTLS")
	}

	// login to server if defined
	if len(config.ServerConfig.Username) > 0 && len(config.ServerConfig.Password) > 0 {
		c.SetUsername(config.ServerConfig.Username)
		c.SetPassword(config.ServerConfig.Password)
		c.SetSMTPAuth(mail.SMTPAuthPlain)
		log.Debug("sendmail: Use Authentication")
	}
	return
}

// add Attachments to mail
func (mt *MailType) attachFiles(m *mail.Msg, maxSize int64) error {
	if maxSize > 0 {
		log.Debugf("attach: File Limit %d", maxSize)
	}
	for _, fn := range mt.Attachments {
		log.Debugf("attach file: %s", fn)
		if maxSize == 0 {
			m.AttachFile(fn, mail.WithFileName(filepath.Base(fn)))
		} else {
			//nolint gosec
			f, oserr := os.Open(fn)
			if oserr != nil {
				errtxt := fmt.Sprintf("attach: Cannot read %s: %v", fn, oserr)
				err := errors.New(errtxt)
				log.Error(errtxt)
				return err
			}
			lr := io.LimitReader(f, maxSize)
			err := m.AttachReader(fn, lr, mail.WithFileName(filepath.Base(fn)))
			if err != nil {
				errtxt := fmt.Sprintf("attach: Cannot attach %s: %v", fn, err)
				log.Error(errtxt)
				return err
			}
		}
	}
	return nil
}

// GetConfig returns current Mail conf
func (config *SendMailConfigType) GetConfig() *SendMailConfigType {
	return config
}
