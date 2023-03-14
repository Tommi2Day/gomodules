package maillib

import (
	"fmt"
	"github.com/tommi2day/gomodules/test"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wneessen/go-mail"

	"github.com/stretchr/testify/assert"
)

const rootUser = "root"
const infoUser = "info"
const rootPass = "testpass"

// const infoPass = "testpass"
const FROM = rootUser + "@" + mailDomain
const TO = infoUser + "@" + mailDomain
const mailDomain = "test.local"
const mailHostname = "mail." + mailDomain

func TestSetMailConfig(t *testing.T) {
	t.Run("Test mailConfig defaults", func(t *testing.T) {
		SetConfig("127.0.0.1", 25, "", "")
		actual := GetConfig()
		assert.Equal(t, "127.0.0.1", actual.Server, "Server default entry not equal")
		assert.Equal(t, 25, actual.Port, "Port entry should be default")
		assert.Empty(t, actual.Username, "Username entry should not set")
	})
	t.Run("Test SetConfig", func(t *testing.T) {
		SetConfig("test.example.com", sslPort, "testuser", "Testpass")
		EnableSSL(true)
		actual := GetConfig()
		assert.Equal(t, "test.example.com", actual.Server, "Server entry not equal")
		assert.Equal(t, sslPort, actual.Port, "Port entry not equal")
		assert.Equal(t, "testuser", actual.Username, "Username entry not equal")
		assert.Equal(t, "Testpass", actual.Password, "Password entry not equal")
		assert.True(t, actual.SSL, "SSL entry not set")
		assert.True(t, actual.SSLinsecure, "Insecure SSL/TLS entry not set")
		assert.False(t, actual.StartTLS, "StartTLS should not set")
	})
	t.Run("Check SetMaxsize", func(t *testing.T) {
		SetMaxSize(2048)
		actual := GetConfig()
		assert.Equal(t, int64(2048), actual.Maxsize)
	})
}

func TestSendMailError(t *testing.T) {
	SetConfig("test.example.com", 25, "testuser", "Testpass")
	t.Run("Send Mail with wrong email", func(t *testing.T) {
		err := SendMail("dummy@local", "root", "TestMail", "My Message")
		assert.Errorf(t, err, "Error: %v", err)
	})
	t.Run("Send Mail with Send Error", func(t *testing.T) {
		err := SendMail("dummy@local", "root@example.com", "TestMail", "My Message")
		assert.Errorf(t, err, "Error: %v", err)
	})
}

func TestMail(t *testing.T) {
	if os.Getenv("SKIP_MAIL") != "" {
		t.Skip("Skipping Mail testing in CI environment")
	}
	var err error
	mailContainer, err = prepareMailContainer()
	require.NoErrorf(t, err, "Mailserver not available: %s", err)
	require.NotNil(t, mailContainer, "Prepare failed")
	defer destroyMailContainer(mailContainer)

	t.Logf("Send tests to %s:%d", mailServer, smtpPort)
	t.Run("Send Mail anonym", func(t *testing.T) {
		SetConfig(mailServer, smtpPort, "", "")
		h := time.Now()
		timeStr := h.Format("15:04:05")
		SetContentType(mail.TypeTextHTML)
		err := SendMail(FROM, TO, "Testmail1", fmt.Sprintf("<html><body>Test at %s</body></html>", timeStr))
		assert.NoErrorf(t, err, "Sendmail anonym returned error %v", err)
	})
	t.Run("Send Mail TLS 25", func(t *testing.T) {
		SetConfig(mailServer, smtpPort, FROM, rootPass)
		EnableTLS(true)
		h := time.Now()
		timeStr := h.Format("15:04:05")
		Cc(FROM)
		err := SendMail(FROM, TO, "Testmail2", fmt.Sprintf("Test at %s", timeStr))
		assert.NoErrorf(t, err, "Sendmail with login returned error %v", err)
	})

	t.Run("Send Mail SSL 465", func(t *testing.T) {
		SetConfig(mailServer, sslPort, FROM, rootPass)
		EnableSSL(true)
		h := time.Now()
		timeStr := h.Format("15:04:05")
		Bcc(FROM)
		Attach([]string{
			test.TestDir + "/tests/ssl/ca.crt",
			test.TestDir + "/tests/sslmail.test.local.crt",
		})
		err := SendMail(FROM, TO, "Testmail3", fmt.Sprintf("Test with ssl at %s", timeStr))
		assert.NoErrorf(t, err, "Sendmail SSL returned error %v", err)
	})
	t.Run("Send Mail TLS 587", func(t *testing.T) {
		SetConfig(mailServer, tlsPort, FROM, rootPass)
		EnableTLS(true)
		SetTimeout(20)
		h := time.Now()
		timeStr := h.Format("15:04:05")
		err := SendMail(FROM, TO, "Testmail4", fmt.Sprintf("Test with tls at %s", timeStr))
		assert.NoErrorf(t, err, "Sendmail TLS returned error %v", err)
	})
}
