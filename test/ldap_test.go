package test

import (
	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/ldap"
	"os"
	"testing"
)

func TestLdap(t *testing.T) {
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	t.Run("Setup Config", func(t *testing.T) {
		ldap.SetConfig("ldap.test", 0, true)
		actual := ldap.GetConfig()
		assert.IsType(t, ldap.ConfigType{}, actual, "Wrong type returned")
		assert.Equal(t, "ldap.test", actual.Server, "Server not equal")
		assert.Equal(t, 636, actual.Port, "with tls=true port should be 636")
	})
}
