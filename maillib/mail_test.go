package maillib

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const FROM = "golibtest@intern.tdressler.net"

func TestSetMailConfig(t *testing.T) {
	t.Run("Test mailConfig defaults", func(t *testing.T) {
		SetConfig("127.0.0.1", 25, "", "", false)
		actual := GetConfig()
		assert.Equal(t, "127.0.0.1", actual.Server, "Server default entry not equal")
		assert.Equal(t, 25, actual.Port, "Port entry should be default")
		assert.Empty(t, actual.Username, "Username entry should not set")
		assert.False(t, actual.TLS, "TLS entry should be false")
	})
	t.Run("Test SetConfig", func(t *testing.T) {
		SetConfig("test.example.com", 587, "testuser", "Testpass", true)
		actual := GetConfig()
		assert.Equal(t, "test.example.com", actual.Server, "Server entry not equal")
		assert.Equal(t, 587, actual.Port, "Port entry not equal")
		assert.Equal(t, "testuser", actual.Username, "Username entry not equal")
		assert.Equal(t, "Testpass", actual.Password, "Password entry not equal")
		assert.True(t, actual.TLS, "TLS entry not set")
	})
	t.Run("Check SetMaxsize", func(t *testing.T) {
		SetMaxSize(2048)
		actual := GetConfig()
		assert.Equal(t, int64(2048), actual.Maxsize)
	})
}

func TestSendMailError(t *testing.T) {
	SetConfig("test.example.com", 25, "testuser", "Testpass", true)
	t.Run("Send Mail with wrong email", func(t *testing.T) {
		err := SendMail("dummy@local", "root", "TestMail", "My Message")
		assert.Errorf(t, err, "Error: %v", err)
	})
	t.Run("Send Mail with Send Error", func(t *testing.T) {
		err := SendMail("dummy@local", "root@example.com", "TestMail", "My Message")
		assert.Errorf(t, err, "Error: %v", err)
	})
}

func TestSendMail(t *testing.T) {
	if os.Getenv("SKIP_MAIL") != "" {
		t.Skip("Skipping Mail testing in CI environment")
	}
	server := os.Getenv("MAIL_SERVER")
	to := os.Getenv("MAIL_TO")
	if len(server) == 0 {
		server = "localhost"
	}
	if len(to) == 0 {
		to = "root@localhost"
	}
	t.Run("Send Mail anonym", func(t *testing.T) {
		SetConfig(server, 25, "", "", false)
		h := time.Now()
		timeStr := h.Format("15:04:05")
		err := SendMail(FROM, to, "Testmail", fmt.Sprintf("Test at %s", timeStr))
		assert.NoErrorf(t, err, "Sendmail returned error %v", err)
	})
	t.Run("Send Mail anonym TLS", func(t *testing.T) {
		SetConfig(server, 465, "", "", true)
		h := time.Now()
		timeStr := h.Format("15:04:05")
		err := SendMail(FROM, to, "Testmail", fmt.Sprintf("Test with ssl at %s", timeStr))
		assert.NoErrorf(t, err, "Sendmail returned error %v", err)
	})
}
