package ldap

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestLdap(t *testing.T) {
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	t.Run("Setup Config", func(t *testing.T) {
		SetConfig("ldap.test", 0, true)
		actual := GetConfig()
		assert.IsType(t, ConfigType{}, actual, "Wrong type returned")
		assert.Equal(t, "ldap.test", actual.Server, "Server not equal")
		assert.Equal(t, 636, actual.Port, "with tls=true port should be 636")
	})
}
