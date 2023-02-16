package test

import (
	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/maillib"
	"os"
	"testing"
)

func TestSetMailConfig(t *testing.T) {
	t.Run("Test MailConfig defaults", func(t *testing.T) {
		maillib.SetMailConfig("", 0, "testuser", "", false)
		assert.Equal(t, "127.0.0.1", maillib.MailConfig.Server, "Server default entry not equal")
		assert.Equal(t, 25, maillib.MailConfig.Port, "Port entry should be default")
		assert.Empty(t, maillib.MailConfig.Username, "Username entry should not set")
		assert.False(t, maillib.MailConfig.Tls, "TLS entry should be false")
	})
	t.Run("Test SetMailConfig", func(t *testing.T) {
		maillib.SetMailConfig("test.example.com", 587, "testuser", "Testpass", true)
		assert.Equal(t, "test.example.com", maillib.MailConfig.Server, "Server entry not equal")
		assert.Equal(t, 587, maillib.MailConfig.Port, "Port entry not equal")
		assert.Equal(t, "testuser", maillib.MailConfig.Username, "Username entry not equal")
		assert.Equal(t, "Testpass", maillib.MailConfig.Password, "Password entry not equal")
		assert.True(t, maillib.MailConfig.Tls, "TLS entry not set")
	})
}

func TestSendMail(t *testing.T) {
	maillib.SetMailConfig("test.example.com", 587, "testuser", "Testpass", true)
	if os.Getenv("SKIP_MAIL") != "" {
		t.Skip("Skipping Mail testing in CI environment")
	}
	t.Run("Send Mail with wrong email", func(t *testing.T) {
		err := maillib.SendMail("golib", []string{"root"}, "TestMail", "My Message", nil, nil, nil)
		assert.Errorf(t, err, "Error: %v", err)
	})
	t.Run("Send Mail with Send Error", func(t *testing.T) {
		err := maillib.SendMail("golib@example.com", []string{"root@example.com"}, "TestMail", "My Message", nil, nil, nil)
		assert.Errorf(t, err, "Error: %v", err)
	})
}
