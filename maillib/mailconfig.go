package maillib

import (
	"crypto/tls"
	"time"
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
}

// NewConfig set Mail server parameter
func NewConfig(server string, port int, username string, password string) *MailConfigType {
	mailConfig := MailConfigType{}
	mailConfig.Server = server
	mailConfig.Port = port
	mailConfig.Username = username
	mailConfig.Password = password
	mailConfig.StartTLS = false
	mailConfig.SSLinsecure = false
	mailConfig.SSL = false
	mailConfig.Timeout = 15 * time.Second
	return &mailConfig
}

// SetTimeout configure max time to connect
func (mailConfig *MailConfigType) SetTimeout(seconds uint) {
	timeout := time.Second * time.Duration(seconds)
	mailConfig.Timeout = timeout
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
}

// EnableTLS allows usage of STARTTLS (e.g. Port 587)
func (mailConfig *MailConfigType) EnableTLS(insecure bool) {
	mailConfig.tlsConfig = &tls.Config{
		//nolint gosec
		InsecureSkipVerify: true,
	}
	mailConfig.StartTLS = true
	mailConfig.SSLinsecure = insecure
	mailConfig.SSL = false
}

// GetConfig returns current Mail conf
func (mailConfig *MailConfigType) GetConfig() *MailConfigType {
	return mailConfig
}
