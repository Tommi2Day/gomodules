package maillib

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/wneessen/go-mail"

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
	AuthMethod  mail.SMTPAuthType
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
	mailConfig.AuthMethod = mail.SMTPAuthPlain
	return &mailConfig
}

// SetTimeout configure max time to connect
func (mailConfig *MailConfigType) SetTimeout(seconds int64) {
	timeout := time.Second * time.Duration(seconds)
	mailConfig.Timeout = timeout
	log.Debugf("mailconfig: Set send timout to %d s", seconds)
}

// EnableSSL allows usage SMTPS Connections (e.g. Port 465)
func (mailConfig *MailConfigType) EnableSSL(insecure bool) {
	mailConfig.tlsConfig = &tls.Config{
		ServerName: mailConfig.Server,
		//nolint gosec
		InsecureSkipVerify: insecure,
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
		ServerName: mailConfig.Server,
		//nolint gosec
		InsecureSkipVerify: insecure,
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

// SetAuthMethod configure Auth Method
func (mailConfig *MailConfigType) SetAuthMethod(method string) {
	var a mail.SMTPAuthType
	switch method {
	case "plain":
		a = mail.SMTPAuthPlain
	case "login":
		a = mail.SMTPAuthLogin
	case "crammd5":
		a = mail.SMTPAuthCramMD5
	case "xoauth2":
		a = mail.SMTPAuthXOAUTH2
	default:
		log.Warnf("mailconfig: Auth Method %s not supported, using plain", method)
		a = mail.SMTPAuthPlain
	}
	mailConfig.AuthMethod = a
	log.Debugf("mailconfig: Set Auth Mech to %s", a)
}

// MailType collects all recipients of a mail and attachment
type MailType struct {
	Attachments       []string
	To                []string
	CC                []string
	Bcc               []string
	From              string
	Subject           string
	Date              time.Time
	TextParts         []string
	ID                uint32
	Signature         string
	SignatureConfig   *MailSignatureConfig
	IsSigned          bool
	SignatureVerified bool
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

// SignMail signs the current mail content with the specified signing configuration
func (mt *MailType) SignMail(config *MailSignatureConfig) error {
	if config == nil {
		return fmt.Errorf("signature config is nil")
	}

	if len(mt.TextParts) == 0 {
		return fmt.Errorf("no content to sign")
	}

	// Join all text parts
	content := strings.Join(mt.TextParts, "\n")

	signature, err := SignMailContent(content, config)
	if err != nil {
		return err
	}

	mt.Signature = signature
	mt.SignatureConfig = config
	mt.IsSigned = true
	log.Debugf("Mail signed with method: %s", config.Method)
	return nil
}

// VerifyMailSignature verifies the signature of the current mail
func (mt *MailType) VerifyMailSignature() (bool, error) {
	if !mt.IsSigned || mt.Signature == "" {
		return false, fmt.Errorf("mail is not signed")
	}

	if mt.SignatureConfig == nil {
		return false, fmt.Errorf("signature config not set")
	}

	// Join all text parts
	content := strings.Join(mt.TextParts, "\n")

	valid, err := VerifyMailSignature(content, mt.Signature, mt.SignatureConfig)
	if err != nil {
		return false, err
	}

	mt.SignatureVerified = valid
	return valid, nil
}

// GetSignatureMethod returns the current signing method
func (mt *MailType) GetSignatureMethod() string {
	if mt.SignatureConfig == nil {
		return ""
	}
	return string(mt.SignatureConfig.Method)
}
