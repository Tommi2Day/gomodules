package maillib

import (
	"crypto/tls"
	"strings"
	"time"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
)

// MailConfigType struct for config properties
type MailConfigType struct {
	Server      string
	Port        int
	Username    string
	Password    string
	SSLinsecure bool
	SSL         bool
	StartTLS    bool
	Timeout     time.Duration
	tlsConfig   *tls.Config
	HELO        string
}

// NewConfig set Mail server parameter
func NewConfig(server string, port int, username string, password string) *MailConfigType {
	hostname := common.GetHostname()
	mailConfig := MailConfigType{}
	mailConfig.Server = server
	mailConfig.Port = port
	mailConfig.Username = username
	mailConfig.Password = password
	mailConfig.StartTLS = false
	mailConfig.SSLinsecure = false
	mailConfig.SSL = false
	mailConfig.Timeout = 15 * time.Second
	mailConfig.HELO = hostname
	return &mailConfig
}

// SetTimeout configure max time to connect
func (mailConfig *MailConfigType) SetTimeout(seconds uint) {
	timeout := time.Second * time.Duration(seconds)
	mailConfig.Timeout = timeout
	log.Debugf("mailconfig: Set send timout to %d s", seconds)
}

// EnableSSL allows usage SMTPS Connections (e.g. Port 465)
func (mailConfig *MailConfigType) EnableSSL(insecure bool) {
	mailConfig.tlsConfig = &tls.Config{
		//nolint gosec
		InsecureSkipVerify: true,
	}
	mailConfig.StartTLS = false
	mailConfig.SSLinsecure = insecure
	mailConfig.SSL = true
	skipVerify := ""
	if insecure {
		skipVerify = "(skip ssl verify:true)"
	}
	log.Debugf("mailconfig: SSL Enabled %s", skipVerify)
}

// EnableTLS allows usage of STARTTLS
func (mailConfig *MailConfigType) EnableTLS(insecure bool) {
	mailConfig.tlsConfig = &tls.Config{
		//nolint gosec
		InsecureSkipVerify: true,
	}
	mailConfig.StartTLS = true
	mailConfig.SSLinsecure = insecure
	mailConfig.SSL = false
	skipVerify := ""
	if insecure {
		skipVerify = "(skip tls verify:true)"
	}
	log.Debugf("mailconfig: TLS Enabled %s", skipVerify)
}

// GetConfig returns current Mail conf
func (mailConfig *MailConfigType) GetConfig() *MailConfigType {
	return mailConfig
}

// SetHELO configure HELO string
func (mailConfig *MailConfigType) SetHELO(helo string) {
	mailConfig.HELO = helo
	log.Debugf("mailconfig: Set HELO to %s", helo)
}

// MailType collects all recipients of a mail and attachment
type MailType struct {
	Attachments []string
	To          []string
	CC          []string
	Bcc         []string
	From        string
	Subject     string
	Date        time.Time
	TextParts   []string
	ID          uint32
}

// NewMail perepare a new Mail Address List
func NewMail(from string, toList string) *MailType {
	al := MailType{}
	al.SetTo(toList)
	al.From = from
	return &al
}

// SetTo sets the list of comma delimited recipents
func (mt *MailType) SetTo(tolist string) {
	mt.To = strings.Split(strings.TrimSpace(tolist), ",")
}

// SetCc sets the list of comma delimited CC'ed recipents
func (mt *MailType) SetCc(cclist string) {
	mt.CC = strings.Split(strings.TrimSpace(cclist), ",")
}

// SetBcc sets the list of comma delimited SetBcc'ed recipents
func (mt *MailType) SetBcc(bcclist string) {
	mt.Bcc = strings.Split(strings.TrimSpace(bcclist), ",")
}

// SetAttach adds list of Attachments (comma delimited full path)
func (mt *MailType) SetAttach(filelist []string) {
	mt.Attachments = filelist
}
